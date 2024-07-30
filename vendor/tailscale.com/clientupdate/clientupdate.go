// Copyright (c) Tailscale Inc & AUTHORS
// SPDX-License-Identifier: BSD-3-Clause

// Package clientupdate implements tailscale client update for all supported
// platforms. This package can be used from both tailscaled and tailscale
// binaries.
package clientupdate

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"tailscale.com/clientupdate/distsign"
	"tailscale.com/types/logger"
	"tailscale.com/util/cmpver"
	"tailscale.com/util/winutil"
	"tailscale.com/version"
	"tailscale.com/version/distro"
)

const (
	CurrentTrack  = ""
	StableTrack   = "stable"
	UnstableTrack = "unstable"
)

func versionToTrack(v string) (string, error) {
	_, rest, ok := strings.Cut(v, ".")
	if !ok {
		return "", fmt.Errorf("malformed version %q", v)
	}
	minorStr, _, ok := strings.Cut(rest, ".")
	if !ok {
		return "", fmt.Errorf("malformed version %q", v)
	}
	minor, err := strconv.Atoi(minorStr)
	if err != nil {
		return "", fmt.Errorf("malformed version %q", v)
	}
	if minor%2 == 0 {
		return "stable", nil
	}
	return "unstable", nil
}

// Arguments contains arguments needed to run an update.
type Arguments struct {
	// Version is the specific version to install.
	// Mutually exclusive with Track.
	Version string
	// Track is the release track to use:
	//
	//   - CurrentTrack will use the latest version from the same track as the
	//     running binary
	//   - StableTrack and UnstableTrack will use the latest versions of the
	//     corresponding tracks
	//
	// Leaving this empty will use Version or fall back to CurrentTrack if both
	// Track and Version are empty.
	Track string
	// Logf is a logger for update progress messages.
	Logf logger.Logf
	// Stdout and Stderr should be used for output instead of os.Stdout and
	// os.Stderr.
	Stdout io.Writer
	Stderr io.Writer
	// Confirm is called when a new version is available and should return true
	// if this new version should be installed. When Confirm returns false, the
	// update is aborted.
	Confirm func(newVer string) bool
	// PkgsAddr is the address of the pkgs server to fetch updates from.
	// Defaults to "https://pkgs.tailscale.com".
	PkgsAddr string
	// ForAutoUpdate should be true when Updater is created in auto-update
	// context. When true, NewUpdater returns an error if it cannot be used for
	// auto-updates (even if Updater.Update field is non-nil).
	ForAutoUpdate bool
}

func (args Arguments) validate() error {
	if args.Confirm == nil {
		return errors.New("missing Confirm callback in Arguments")
	}
	if args.Logf == nil {
		return errors.New("missing Logf callback in Arguments")
	}
	if args.Version != "" && args.Track != "" {
		return fmt.Errorf("only one of Version(%q) or Track(%q) can be set", args.Version, args.Track)
	}
	switch args.Track {
	case StableTrack, UnstableTrack, CurrentTrack:
		// All valid values.
	default:
		return fmt.Errorf("unsupported track %q", args.Track)
	}
	return nil
}

type Updater struct {
	Arguments
	// Update is a platform-specific method that updates the installation. May be
	// nil (not all platforms support updates from within Tailscale).
	Update func() error
}

func NewUpdater(args Arguments) (*Updater, error) {
	up := Updater{
		Arguments: args,
	}
	if up.Stdout == nil {
		up.Stdout = os.Stdout
	}
	if up.Stderr == nil {
		up.Stderr = os.Stderr
	}
	var canAutoUpdate bool
	up.Update, canAutoUpdate = up.getUpdateFunction()
	if up.Update == nil {
		return nil, errors.ErrUnsupported
	}
	if args.ForAutoUpdate && !canAutoUpdate {
		return nil, errors.ErrUnsupported
	}
	if up.Track == CurrentTrack {
		switch {
		case up.Version != "":
			var err error
			up.Track, err = versionToTrack(args.Version)
			if err != nil {
				return nil, err
			}
		case version.IsUnstableBuild():
			up.Track = UnstableTrack
		default:
			up.Track = StableTrack
		}
	}
	if up.Arguments.PkgsAddr == "" {
		up.Arguments.PkgsAddr = "https://pkgs.tailscale.com"
	}
	return &up, nil
}

type updateFunction func() error

func (up *Updater) getUpdateFunction() (fn updateFunction, canAutoUpdate bool) {
	switch runtime.GOOS {
	case "windows":
		return up.updateWindows, true
	case "linux":
		switch distro.Get() {
		case distro.Synology:
			// Synology updates use our own pkgs.tailscale.com instead of the
			// Synology Package Center. We should eventually get to a regular
			// release cadence with Synology Package Center and use their
			// auto-update mechanism.
			return up.updateSynology, false
		case distro.Debian: // includes Ubuntu
			return up.updateDebLike, true
		case distro.Arch:
			if up.archPackageInstalled() {
				// Arch update func just prints a message about how to update,
				// it doesn't support auto-updates.
				return up.updateArchLike, false
			}
			return up.updateLinuxBinary, true
		case distro.Alpine:
			return up.updateAlpineLike, true
		case distro.Unraid:
			return up.updateUnraid, true
		case distro.QNAP:
			return up.updateQNAP, true
		}
		switch {
		case haveExecutable("pacman"):
			if up.archPackageInstalled() {
				// Arch update func just prints a message about how to update,
				// it doesn't support auto-updates.
				return up.updateArchLike, false
			}
			return up.updateLinuxBinary, true
		case haveExecutable("apt-get"): // TODO(awly): add support for "apt"
			// The distro.Debian switch case above should catch most apt-based
			// systems, but add this fallback just in case.
			return up.updateDebLike, true
		case haveExecutable("dnf"):
			return up.updateFedoraLike("dnf"), true
		case haveExecutable("yum"):
			return up.updateFedoraLike("yum"), true
		case haveExecutable("apk"):
			return up.updateAlpineLike, true
		}
		// If nothing matched, fall back to tarball updates.
		if up.Update == nil {
			return up.updateLinuxBinary, true
		}
	case "darwin":
		switch {
		case version.IsMacAppStore():
			// App store update func just opens the store page, it doesn't
			// support auto-updates.
			return up.updateMacAppStore, false
		case version.IsMacSysExt():
			// Macsys update func kicks off Sparkle. Auto-updates are done by
			// Sparkle.
			return up.updateMacSys, false
		default:
			return nil, false
		}
	case "freebsd":
		return up.updateFreeBSD, true
	}
	return nil, false
}

// CanAutoUpdate reports whether auto-updating via the clientupdate package
// is supported for the current os/distro.
func CanAutoUpdate() bool {
	_, canAutoUpdate := (&Updater{}).getUpdateFunction()
	return canAutoUpdate
}

// Update runs a single update attempt using the platform-specific mechanism.
//
// On Windows, this copies the calling binary and re-executes it to apply the
// update. The calling binary should handle an "update" subcommand and call
// this function again for the re-executed binary to proceed.
func Update(args Arguments) error {
	if err := args.validate(); err != nil {
		return err
	}
	up, err := NewUpdater(args)
	if err != nil {
		return err
	}
	return up.Update()
}

func (up *Updater) confirm(ver string) bool {
	switch cmpver.Compare(version.Short(), ver) {
	case 0:
		up.Logf("already running %v version %v; no update needed", up.Track, ver)
		return false
	case 1:
		up.Logf("installed %v version %v is newer than the latest available version %v; no update needed", up.Track, version.Short(), ver)
		return false
	}
	if up.Confirm != nil {
		return up.Confirm(ver)
	}
	return true
}

const synoinfoConfPath = "/etc/synoinfo.conf"

func (up *Updater) updateSynology() error {
	if up.Version != "" {
		return errors.New("installing a specific version on Synology is not supported")
	}
	if err := requireRoot(); err != nil {
		return err
	}

	// Get the latest version and list of SPKs from pkgs.tailscale.com.
	dsmVersion := distro.DSMVersion()
	osName := fmt.Sprintf("dsm%d", dsmVersion)
	arch, err := synoArch(runtime.GOARCH, synoinfoConfPath)
	if err != nil {
		return err
	}
	latest, err := latestPackages(up.Track)
	if err != nil {
		return err
	}
	spkName := latest.SPKs[osName][arch]
	if spkName == "" {
		return fmt.Errorf("cannot find Synology package for os=%s arch=%s, please report a bug with your device model", osName, arch)
	}

	if !up.confirm(latest.SPKsVersion) {
		return nil
	}

	up.cleanupOldDownloads(filepath.Join(os.TempDir(), "tailscale-update*", "*.spk"))
	// Download the SPK into a temporary directory.
	spkDir, err := os.MkdirTemp("", "tailscale-update")
	if err != nil {
		return err
	}
	pkgsPath := fmt.Sprintf("%s/%s", up.Track, spkName)
	spkPath := filepath.Join(spkDir, path.Base(pkgsPath))
	if err := up.downloadURLToFile(pkgsPath, spkPath); err != nil {
		return err
	}

	// Install the SPK. Run via nohup to allow install to succeed when we're
	// connected over tailscale ssh and this parent process dies. Otherwise, if
	// you abort synopkg install mid-way, tailscaled is not restarted.
	cmd := exec.Command("nohup", "synopkg", "install", spkPath)
	// Don't attach cmd.Stdout to Stdout because nohup will redirect that into
	// nohup.out file. synopkg doesn't have any progress output anyway, it just
	// spits out a JSON result when done.
	out, err := cmd.CombinedOutput()
	if err != nil {
		if dsmVersion == 6 && bytes.Contains(out, []byte("error = [290]")) {
			return fmt.Errorf("synopkg install failed: %w\noutput:\n%s\nplease make sure that packages from 'Any publisher' are allowed in the Package Center (Package Center -> Settings -> Trust Level -> Any publisher)", err, out)
		}
		return fmt.Errorf("synopkg install failed: %w\noutput:\n%s", err, out)
	}
	if dsmVersion == 6 {
		// DSM6 does not automatically restart the package on install. Do it
		// manually.
		cmd := exec.Command("nohup", "synopkg", "start", "Tailscale")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("synopkg start failed: %w\noutput:\n%s", err, out)
		}
	}
	return nil
}

// synoArch returns the Synology CPU architecture matching one of the SPK
// architectures served from pkgs.tailscale.com.
func synoArch(goArch, synoinfoPath string) (string, error) {
	// Most Synology boxes just use a different arch name from GOARCH.
	arch := map[string]string{
		"amd64": "x86_64",
		"386":   "i686",
		"arm64": "armv8",
	}[goArch]

	if arch == "" {
		// Here's the fun part, some older ARM boxes require you to use SPKs
		// specifically for their CPU. See
		// https://github.com/SynoCommunity/spksrc/wiki/Synology-and-SynoCommunity-Package-Architectures
		// for a complete list.
		//
		// Some CPUs will map to neither this list nor the goArch map above, and we
		// don't have SPKs for them.
		cpu, err := parseSynoinfo(synoinfoPath)
		if err != nil {
			return "", fmt.Errorf("failed to get CPU architecture: %w", err)
		}
		switch cpu {
		case "88f6281", "88f6282", "hi3535", "alpine", "armada370",
			"armada375", "armada38x", "armadaxp", "comcerto2k", "monaco":
			arch = cpu
		default:
			return "", fmt.Errorf("unsupported Synology CPU architecture %q (Go arch %q), please report a bug at https://github.com/tailscale/tailscale/issues/new/choose", cpu, goArch)
		}
	}
	return arch, nil
}

func parseSynoinfo(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	// Look for a line like:
	// unique="synology_88f6282_413j"
	// Extract the CPU in the middle (88f6282 in the above example).
	s := bufio.NewScanner(f)
	for s.Scan() {
		l := s.Text()
		if !strings.HasPrefix(l, "unique=") {
			continue
		}
		parts := strings.SplitN(l, "_", 3)
		if len(parts) != 3 {
			return "", fmt.Errorf(`malformed %q: found %q, expected format like 'unique="synology_$cpu_$model'`, path, l)
		}
		return parts[1], nil
	}
	return "", fmt.Errorf(`missing "unique=" field in %q`, path)
}

func (up *Updater) updateDebLike() error {
	if err := requireRoot(); err != nil {
		return err
	}
	if err := exec.Command("dpkg", "--status", "tailscale").Run(); err != nil && isExitError(err) {
		// Tailscale was not installed via apt, update via tarball download
		// instead.
		return up.updateLinuxBinary()
	}
	ver, err := requestedTailscaleVersion(up.Version, up.Track)
	if err != nil {
		return err
	}
	if !up.confirm(ver) {
		return nil
	}

	if updated, err := updateDebianAptSourcesList(up.Track); err != nil {
		return err
	} else if updated {
		up.Logf("Updated %s to use the %s track", aptSourcesFile, up.Track)
	}

	cmd := exec.Command("apt-get", "update",
		// Only update the tailscale repo, not the other ones, treating
		// the tailscale.list file as the main "sources.list" file.
		"-o", "Dir::Etc::SourceList=sources.list.d/tailscale.list",
		// Disable the "sources.list.d" directory:
		"-o", "Dir::Etc::SourceParts=-",
		// Don't forget about packages in the other repos just because
		// we're not updating them:
		"-o", "APT::Get::List-Cleanup=0",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("apt-get update failed: %w; output:\n%s", err, out)
	}

	for i := 0; i < 2; i++ {
		out, err := exec.Command("apt-get", "install", "--yes", "--allow-downgrades", "tailscale="+ver).CombinedOutput()
		if err != nil {
			if !bytes.Contains(out, []byte(`dpkg was interrupted`)) {
				return fmt.Errorf("apt-get install failed: %w; output:\n%s", err, out)
			}
			up.Logf("apt-get install failed: %s; output:\n%s", err, out)
			up.Logf("running dpkg --configure tailscale")
			out, err = exec.Command("dpkg", "--force-confdef,downgrade", "--configure", "tailscale").CombinedOutput()
			if err != nil {
				return fmt.Errorf("dpkg --configure tailscale failed: %w; output:\n%s", err, out)
			}
			continue
		}
		break
	}

	return nil
}

const aptSourcesFile = "/etc/apt/sources.list.d/tailscale.list"

// updateDebianAptSourcesList updates the /etc/apt/sources.list.d/tailscale.list
// file to make sure it has the provided track (stable or unstable) in it.
//
// If it already has the right track (including containing both stable and
// unstable), it does nothing.
func updateDebianAptSourcesList(dstTrack string) (rewrote bool, err error) {
	was, err := os.ReadFile(aptSourcesFile)
	if err != nil {
		return false, err
	}
	newContent, err := updateDebianAptSourcesListBytes(was, dstTrack)
	if err != nil {
		return false, err
	}
	if bytes.Equal(was, newContent) {
		return false, nil
	}
	return true, os.WriteFile(aptSourcesFile, newContent, 0644)
}

func updateDebianAptSourcesListBytes(was []byte, dstTrack string) (newContent []byte, err error) {
	trackURLPrefix := []byte("https://pkgs.tailscale.com/" + dstTrack + "/")
	var buf bytes.Buffer
	var changes int
	bs := bufio.NewScanner(bytes.NewReader(was))
	hadCorrect := false
	commentLine := regexp.MustCompile(`^\s*\#`)
	pkgsURL := regexp.MustCompile(`\bhttps://pkgs\.tailscale\.com/((un)?stable)/`)
	for bs.Scan() {
		line := bs.Bytes()
		if !commentLine.Match(line) {
			line = pkgsURL.ReplaceAllFunc(line, func(m []byte) []byte {
				if bytes.Equal(m, trackURLPrefix) {
					hadCorrect = true
				} else {
					changes++
				}
				return trackURLPrefix
			})
		}
		buf.Write(line)
		buf.WriteByte('\n')
	}
	if hadCorrect || (changes == 1 && bytes.Equal(bytes.TrimSpace(was), bytes.TrimSpace(buf.Bytes()))) {
		// Unchanged or close enough.
		return was, nil
	}
	if changes != 1 {
		// No changes, or an unexpected number of changes (what?). Bail.
		// They probably editted it by hand and we don't know what to do.
		return nil, fmt.Errorf("unexpected/unsupported %s contents", aptSourcesFile)
	}
	return buf.Bytes(), nil
}

func (up *Updater) archPackageInstalled() bool {
	err := exec.Command("pacman", "--query", "tailscale").Run()
	return err == nil
}

func (up *Updater) updateArchLike() error {
	// Arch maintainer asked us not to implement "tailscale update" or
	// auto-updates on Arch-based distros:
	// https://github.com/tailscale/tailscale/issues/6995#issuecomment-1687080106
	return errors.New(`individual package updates are not supported on Arch-based distros, only full-system updates are: https://wiki.archlinux.org/title/System_maintenance#Partial_upgrades_are_unsupported.
you can use "pacman --sync --refresh --sysupgrade" or "pacman -Syu" to upgrade the system, including Tailscale.`)
}

const yumRepoConfigFile = "/etc/yum.repos.d/tailscale.repo"

// updateFedoraLike updates tailscale on any distros in the Fedora family,
// specifically anything that uses "dnf" or "yum" package managers. The actual
// package manager is passed via packageManager.
func (up *Updater) updateFedoraLike(packageManager string) func() error {
	return func() (err error) {
		if err := requireRoot(); err != nil {
			return err
		}
		if err := exec.Command(packageManager, "info", "--installed", "tailscale").Run(); err != nil && isExitError(err) {
			// Tailscale was not installed via yum/dnf, update via tarball
			// download instead.
			return up.updateLinuxBinary()
		}
		defer func() {
			if err != nil {
				err = fmt.Errorf(`%w; you can try updating using "%s upgrade tailscale"`, err, packageManager)
			}
		}()

		ver, err := requestedTailscaleVersion(up.Version, up.Track)
		if err != nil {
			return err
		}
		if !up.confirm(ver) {
			return nil
		}

		if updated, err := updateYUMRepoTrack(yumRepoConfigFile, up.Track); err != nil {
			return err
		} else if updated {
			up.Logf("Updated %s to use the %s track", yumRepoConfigFile, up.Track)
		}

		cmd := exec.Command(packageManager, "install", "--assumeyes", fmt.Sprintf("tailscale-%s-1", ver))
		cmd.Stdout = up.Stdout
		cmd.Stderr = up.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
		return nil
	}
}

// updateYUMRepoTrack updates the repoFile file to make sure it has the
// provided track (stable or unstable) in it.
func updateYUMRepoTrack(repoFile, dstTrack string) (rewrote bool, err error) {
	was, err := os.ReadFile(repoFile)
	if err != nil {
		return false, err
	}

	urlRe := regexp.MustCompile(`^(baseurl|gpgkey)=https://pkgs\.tailscale\.com/(un)?stable/`)
	urlReplacement := fmt.Sprintf("$1=https://pkgs.tailscale.com/%s/", dstTrack)

	s := bufio.NewScanner(bytes.NewReader(was))
	newContent := bytes.NewBuffer(make([]byte, 0, len(was)))
	for s.Scan() {
		line := s.Text()
		// Handle repo section name, like "[tailscale-stable]".
		if len(line) > 0 && line[0] == '[' {
			if !strings.HasPrefix(line, "[tailscale-") {
				return false, fmt.Errorf("%q does not look like a tailscale repo file, it contains an unexpected %q section", repoFile, line)
			}
			fmt.Fprintf(newContent, "[tailscale-%s]\n", dstTrack)
			continue
		}
		// Update the track mentioned in repo name.
		if strings.HasPrefix(line, "name=") {
			fmt.Fprintf(newContent, "name=Tailscale %s\n", dstTrack)
			continue
		}
		// Update the actual repo URLs.
		if strings.HasPrefix(line, "baseurl=") || strings.HasPrefix(line, "gpgkey=") {
			fmt.Fprintln(newContent, urlRe.ReplaceAllString(line, urlReplacement))
			continue
		}
		fmt.Fprintln(newContent, line)
	}
	if bytes.Equal(was, newContent.Bytes()) {
		return false, nil
	}
	return true, os.WriteFile(repoFile, newContent.Bytes(), 0644)
}

func (up *Updater) updateAlpineLike() (err error) {
	if up.Version != "" {
		return errors.New("installing a specific version on Alpine-based distros is not supported")
	}
	if err := requireRoot(); err != nil {
		return err
	}
	if err := exec.Command("apk", "info", "--installed", "tailscale").Run(); err != nil && isExitError(err) {
		// Tailscale was not installed via apk, update via tarball download
		// instead.
		return up.updateLinuxBinary()
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf(`%w; you can try updating using "apk upgrade tailscale"`, err)
		}
	}()

	out, err := exec.Command("apk", "update").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed refresh apk repository indexes: %w, output:\n%s", err, out)
	}
	out, err = exec.Command("apk", "info", "tailscale").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed checking apk for latest tailscale version: %w, output:\n%s", err, out)
	}
	ver, err := parseAlpinePackageVersion(out)
	if err != nil {
		return fmt.Errorf(`failed to parse latest version from "apk info tailscale": %w`, err)
	}
	if !up.confirm(ver) {
		return nil
	}

	cmd := exec.Command("apk", "upgrade", "tailscale")
	cmd.Stdout = up.Stdout
	cmd.Stderr = up.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed tailscale update using apk: %w", err)
	}
	return nil
}

func parseAlpinePackageVersion(out []byte) (string, error) {
	s := bufio.NewScanner(bytes.NewReader(out))
	for s.Scan() {
		// The line should look like this:
		// tailscale-1.44.2-r0 description:
		line := strings.TrimSpace(s.Text())
		if !strings.HasPrefix(line, "tailscale-") {
			continue
		}
		parts := strings.SplitN(line, "-", 3)
		if len(parts) < 3 {
			return "", fmt.Errorf("malformed info line: %q", line)
		}
		return parts[1], nil
	}
	return "", errors.New("tailscale version not found in output")
}

func (up *Updater) updateMacSys() error {
	return errors.New("NOTREACHED: On MacSys builds, `tailscale update` is handled in Swift to launch the GUI updater")
}

func (up *Updater) updateMacAppStore() error {
	// We can't trigger the update via App Store from the sandboxed app. At
	// most, we can open the App Store page for them.
	up.Logf("Please use the App Store to update Tailscale.\nConsider enabling Automatic Updates in the App Store Settings, if you haven't already.\nOpening the Tailscale app page...")

	out, err := exec.Command("open", "https://apps.apple.com/us/app/tailscale/id1475387142").CombinedOutput()
	if err != nil {
		return fmt.Errorf("can't open the Tailscale page in App Store: %w, output:\n%s", err, string(out))
	}
	return nil
}

const (
	// winMSIEnv is the environment variable that, if set, is the MSI file for
	// the update command to install. It's passed like this so we can stop the
	// tailscale.exe process from running before the msiexec process runs and
	// tries to overwrite ourselves.
	winMSIEnv = "TS_UPDATE_WIN_MSI"
	// winExePathEnv is the environment variable that is set along with
	// winMSIEnv and carries the full path of the calling tailscale.exe binary.
	// It is used to re-launch the GUI process (tailscale-ipn.exe) after
	// install is complete.
	winExePathEnv = "TS_UPDATE_WIN_EXE_PATH"
)

var (
	verifyAuthenticode func(string) error // set non-nil only on Windows
	markTempFileFunc   func(string) error // set non-nil only on Windows
)

func (up *Updater) updateWindows() error {
	if msi := os.Getenv(winMSIEnv); msi != "" {
		// stdout/stderr from this part of the install could be lost since the
		// parent tailscaled is replaced. Create a temp log file to have some
		// output to debug with in case update fails.
		close, err := up.switchOutputToFile()
		if err != nil {
			up.Logf("failed to create log file for installation: %v; proceeding with existing outputs", err)
		} else {
			defer close.Close()
		}

		up.Logf("installing %v ...", msi)
		if err := up.installMSI(msi); err != nil {
			up.Logf("MSI install failed: %v", err)
			return err
		}

		up.Logf("success.")
		return nil
	}

	if !winutil.IsCurrentProcessElevated() {
		return errors.New(`update must be run as Administrator

you can run the command prompt as Administrator one of these ways:
* right-click cmd.exe, select 'Run as administrator'
* press Windows+x, then press a
* press Windows+r, type in "cmd", then press Ctrl+Shift+Enter`)
	}
	ver, err := requestedTailscaleVersion(up.Version, up.Track)
	if err != nil {
		return err
	}
	arch := runtime.GOARCH
	if arch == "386" {
		arch = "x86"
	}
	if !up.confirm(ver) {
		return nil
	}

	tsDir := filepath.Join(os.Getenv("ProgramData"), "Tailscale")
	msiDir := filepath.Join(tsDir, "MSICache")
	if fi, err := os.Stat(tsDir); err != nil {
		return fmt.Errorf("expected %s to exist, got stat error: %w", tsDir, err)
	} else if !fi.IsDir() {
		return fmt.Errorf("expected %s to be a directory; got %v", tsDir, fi.Mode())
	}
	if err := os.MkdirAll(msiDir, 0700); err != nil {
		return err
	}
	up.cleanupOldDownloads(filepath.Join(msiDir, "*.msi"))
	pkgsPath := fmt.Sprintf("%s/tailscale-setup-%s-%s.msi", up.Track, ver, arch)
	msiTarget := filepath.Join(msiDir, path.Base(pkgsPath))
	if err := up.downloadURLToFile(pkgsPath, msiTarget); err != nil {
		return err
	}

	up.Logf("verifying MSI authenticode...")
	if err := verifyAuthenticode(msiTarget); err != nil {
		return fmt.Errorf("authenticode verification of %s failed: %w", msiTarget, err)
	}
	up.Logf("authenticode verification succeeded")

	up.Logf("making tailscale.exe copy to switch to...")
	up.cleanupOldDownloads(filepath.Join(os.TempDir(), "tailscale-updater-*.exe"))
	selfOrig, selfCopy, err := makeSelfCopy()
	if err != nil {
		return err
	}
	defer os.Remove(selfCopy)
	up.Logf("running tailscale.exe copy for final install...")

	cmd := exec.Command(selfCopy, "update")
	cmd.Env = append(os.Environ(), winMSIEnv+"="+msiTarget, winExePathEnv+"="+selfOrig)
	cmd.Stdout = up.Stderr
	cmd.Stderr = up.Stderr
	cmd.Stdin = os.Stdin
	if err := cmd.Start(); err != nil {
		return err
	}
	// Once it's started, exit ourselves, so the binary is free
	// to be replaced.
	os.Exit(0)
	panic("unreachable")
}

func (up *Updater) switchOutputToFile() (io.Closer, error) {
	var logFilePath string
	exePath, err := os.Executable()
	if err != nil {
		logFilePath = filepath.Join(os.TempDir(), "tailscale-updater.log")
	} else {
		logFilePath = strings.TrimSuffix(exePath, ".exe") + ".log"
	}

	up.Logf("writing update output to %q", logFilePath)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return nil, err
	}

	up.Logf = func(m string, args ...any) {
		fmt.Fprintf(logFile, m+"\n", args...)
	}
	up.Stdout = logFile
	up.Stderr = logFile
	return logFile, nil
}

func (up *Updater) installMSI(msi string) error {
	var err error
	for tries := 0; tries < 2; tries++ {
		cmd := exec.Command("msiexec.exe", "/i", filepath.Base(msi), "/quiet", "/promptrestart", "/qn")
		cmd.Dir = filepath.Dir(msi)
		cmd.Stdout = up.Stdout
		cmd.Stderr = up.Stderr
		cmd.Stdin = os.Stdin
		err = cmd.Run()
		if err == nil {
			break
		}
		up.Logf("Install attempt failed: %v", err)
		uninstallVersion := version.Short()
		if v := os.Getenv("TS_DEBUG_UNINSTALL_VERSION"); v != "" {
			uninstallVersion = v
		}
		// Assume it's a downgrade, which msiexec won't permit. Uninstall our current version first.
		up.Logf("Uninstalling current version %q for downgrade...", uninstallVersion)
		cmd = exec.Command("msiexec.exe", "/x", msiUUIDForVersion(uninstallVersion), "/norestart", "/qn")
		cmd.Stdout = up.Stdout
		cmd.Stderr = up.Stderr
		cmd.Stdin = os.Stdin
		err = cmd.Run()
		up.Logf("msiexec uninstall: %v", err)
	}
	return err
}

// cleanupOldDownloads removes all files matching glob (see filepath.Glob).
// Only regular files are removed, so the glob must match specific files and
// not directories.
func (up *Updater) cleanupOldDownloads(glob string) {
	matches, err := filepath.Glob(glob)
	if err != nil {
		up.Logf("cleaning up old downloads: %v", err)
		return
	}
	for _, m := range matches {
		s, err := os.Lstat(m)
		if err != nil {
			up.Logf("cleaning up old downloads: %v", err)
			continue
		}
		if !s.Mode().IsRegular() {
			continue
		}
		if err := os.Remove(m); err != nil {
			up.Logf("cleaning up old downloads: %v", err)
		}
	}
}

func msiUUIDForVersion(ver string) string {
	arch := runtime.GOARCH
	if arch == "386" {
		arch = "x86"
	}
	track, err := versionToTrack(ver)
	if err != nil {
		track = UnstableTrack
	}
	msiURL := fmt.Sprintf("https://pkgs.tailscale.com/%s/tailscale-setup-%s-%s.msi", track, ver, arch)
	return "{" + strings.ToUpper(uuid.NewSHA1(uuid.NameSpaceURL, []byte(msiURL)).String()) + "}"
}

func makeSelfCopy() (origPathExe, tmpPathExe string, err error) {
	selfExe, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	f, err := os.Open(selfExe)
	if err != nil {
		return "", "", err
	}
	defer f.Close()
	f2, err := os.CreateTemp("", "tailscale-updater-*.exe")
	if err != nil {
		return "", "", err
	}
	if f := markTempFileFunc; f != nil {
		if err := f(f2.Name()); err != nil {
			return "", "", err
		}
	}
	if _, err := io.Copy(f2, f); err != nil {
		f2.Close()
		return "", "", err
	}
	return selfExe, f2.Name(), f2.Close()
}

func (up *Updater) downloadURLToFile(pathSrc, fileDst string) (ret error) {
	c, err := distsign.NewClient(up.Logf, up.PkgsAddr)
	if err != nil {
		return err
	}
	return c.Download(context.Background(), pathSrc, fileDst)
}

func (up *Updater) updateFreeBSD() (err error) {
	if up.Version != "" {
		return errors.New("installing a specific version on FreeBSD is not supported")
	}
	if err := requireRoot(); err != nil {
		return err
	}
	if err := exec.Command("pkg", "query", "%n", "tailscale").Run(); err != nil && isExitError(err) {
		// Tailscale was not installed via pkg and we don't pre-compile
		// binaries for it.
		return errors.New("Tailscale was not installed via pkg, binary updates on FreeBSD are not supported; please reinstall Tailscale using pkg or update manually")
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf(`%w; you can try updating using "pkg upgrade tailscale"`, err)
		}
	}()

	out, err := exec.Command("pkg", "update").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed refresh pkg repository indexes: %w, output:\n%s", err, out)
	}
	out, err = exec.Command("pkg", "rquery", "%v", "tailscale").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed checking pkg for latest tailscale version: %w, output:\n%s", err, out)
	}
	ver := string(bytes.TrimSpace(out))
	if !up.confirm(ver) {
		return nil
	}

	cmd := exec.Command("pkg", "upgrade", "-y", "tailscale")
	cmd.Stdout = up.Stdout
	cmd.Stderr = up.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed tailscale update using pkg: %w", err)
	}

	// pkg does not automatically restart services after upgrade.
	out, err = exec.Command("service", "tailscaled", "restart").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to restart tailscaled after update: %w, output:\n%s", err, out)
	}
	return nil
}

func (up *Updater) updateLinuxBinary() error {
	// Root is needed to overwrite binaries and restart systemd unit.
	if err := requireRoot(); err != nil {
		return err
	}
	ver, err := requestedTailscaleVersion(up.Version, up.Track)
	if err != nil {
		return err
	}
	if !up.confirm(ver) {
		return nil
	}

	dlPath, err := up.downloadLinuxTarball(ver)
	if err != nil {
		return err
	}
	up.Logf("Extracting %q", dlPath)
	if err := up.unpackLinuxTarball(dlPath); err != nil {
		return err
	}
	if err := os.Remove(dlPath); err != nil {
		up.Logf("failed to clean up %q: %v", dlPath, err)
	}
	if err := restartSystemdUnit(context.Background()); err != nil {
		if errors.Is(err, errors.ErrUnsupported) {
			up.Logf("Tailscale binaries updated successfully.\nPlease restart tailscaled to finish the update.")
		} else {
			up.Logf("Tailscale binaries updated successfully, but failed to restart tailscaled: %s.\nPlease restart tailscaled to finish the update.", err)
		}
	} else {
		up.Logf("Success")
	}

	return nil
}

func (up *Updater) downloadLinuxTarball(ver string) (string, error) {
	dlDir, err := os.UserCacheDir()
	if err != nil {
		dlDir = os.TempDir()
	}
	dlDir = filepath.Join(dlDir, "tailscale-update")
	if err := os.MkdirAll(dlDir, 0700); err != nil {
		return "", err
	}
	pkgsPath := fmt.Sprintf("%s/tailscale_%s_%s.tgz", up.Track, ver, runtime.GOARCH)
	dlPath := filepath.Join(dlDir, path.Base(pkgsPath))
	if err := up.downloadURLToFile(pkgsPath, dlPath); err != nil {
		return "", err
	}
	return dlPath, nil
}

func (up *Updater) unpackLinuxTarball(path string) error {
	tailscale, tailscaled, err := binaryPaths()
	if err != nil {
		return err
	}
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	gr, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gr.Close()
	tr := tar.NewReader(gr)
	files := make(map[string]int)
	wantFiles := map[string]int{
		"tailscale":  1,
		"tailscaled": 1,
	}
	for {
		th, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed extracting %q: %w", path, err)
		}
		// TODO(awly): try to also extract tailscaled.service. The tricky part
		// is fixing up binary paths in that file if they differ from where
		// local tailscale/tailscaled are installed. Also, this may not be a
		// systemd distro.
		switch filepath.Base(th.Name) {
		case "tailscale":
			files["tailscale"]++
			if err := writeFile(tr, tailscale+".new", 0755); err != nil {
				return fmt.Errorf("failed extracting the new tailscale binary from %q: %w", path, err)
			}
		case "tailscaled":
			files["tailscaled"]++
			if err := writeFile(tr, tailscaled+".new", 0755); err != nil {
				return fmt.Errorf("failed extracting the new tailscaled binary from %q: %w", path, err)
			}
		}
	}
	if !maps.Equal(files, wantFiles) {
		return fmt.Errorf("%q has missing or duplicate files: got %v, want %v", path, files, wantFiles)
	}

	// Only place the files in final locations after everything extracted correctly.
	if err := os.Rename(tailscale+".new", tailscale); err != nil {
		return err
	}
	up.Logf("Updated %s", tailscale)
	if err := os.Rename(tailscaled+".new", tailscaled); err != nil {
		return err
	}
	up.Logf("Updated %s", tailscaled)
	return nil
}

func (up *Updater) updateQNAP() (err error) {
	if up.Version != "" {
		return errors.New("installing a specific version on QNAP is not supported")
	}
	if err := requireRoot(); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf(`%w; you can try updating using "qpkg_cli --add Tailscale"`, err)
		}
	}()

	out, err := exec.Command("qpkg_cli", "--upgradable", "Tailscale").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check if Tailscale is upgradable using qpkg_cli: %w, output: %q", err, out)
	}

	// Output should look like this:
	//
	// $ qpkg_cli -G Tailscale
	// [Tailscale]
	// upgradeStatus = 1
	statusRe := regexp.MustCompile(`upgradeStatus = (\d)`)
	m := statusRe.FindStringSubmatch(string(out))
	if len(m) < 2 {
		return fmt.Errorf("failed to check if Tailscale is upgradable using qpkg_cli, output: %q", out)
	}
	status, err := strconv.Atoi(m[1])
	if err != nil {
		return fmt.Errorf("cannot parse upgradeStatus from qpkg_cli output %q: %w", out, err)
	}
	// Possible status values:
	//  0:can upgrade
	//  1:can not upgrade
	//  2:error
	//  3:can not get rss information
	//  4:qpkg not found
	//  5:qpkg not installed
	//
	// We want status 0.
	switch status {
	case 0: // proceed with upgrade
	case 1:
		up.Logf("no update available")
		return nil
	case 2, 3, 4:
		return fmt.Errorf("failed to check update status with qpkg_cli (upgradeStatus = %d)", status)
	case 5:
		return errors.New("Tailscale was not found in the QNAP App Center")
	default:
		return fmt.Errorf("failed to check update status with qpkg_cli (upgradeStatus = %d)", status)
	}

	// There doesn't seem to be a way to fetch what the available upgrade
	// version is. Use the generic "latest" version in confirmation prompt.
	if up.Confirm != nil && !up.Confirm("latest") {
		return nil
	}

	up.Logf("c2n: running qpkg_cli --add Tailscale")
	cmd := exec.Command("qpkg_cli", "--add", "Tailscale")
	cmd.Stdout = up.Stdout
	cmd.Stderr = up.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed tailscale update using qpkg_cli: %w", err)
	}
	return nil
}

func (up *Updater) updateUnraid() (err error) {
	if up.Version != "" {
		return errors.New("installing a specific version on Unraid is not supported")
	}
	if err := requireRoot(); err != nil {
		return err
	}

	defer func() {
		if err != nil {
			err = fmt.Errorf(`%w; you can try updating using "plugin check tailscale.plg && plugin update tailscale.plg"`, err)
		}
	}()

	// We need to run `plugin check` for the latest tailscale.plg to get
	// downloaded. Unfortunately, the output of this command does not contain
	// the latest tailscale version available. So we'll parse the downloaded
	// tailscale.plg file manually below.
	out, err := exec.Command("plugin", "check", "tailscale.plg").CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to check if Tailscale plugin is upgradable: %w, output: %q", err, out)
	}

	// Note: 'plugin check' downloads plugins to /tmp/plugins.
	// The installed .plg files are in /boot/config/plugins/, but the pending
	// ones are in /tmp/plugins. We should parse the pending file downloaded by
	// 'plugin check'.
	latest, err := parseUnraidPluginVersion("/tmp/plugins/tailscale.plg")
	if err != nil {
		return fmt.Errorf("failed to find latest Tailscale version in /boot/config/plugins/tailscale.plg: %w", err)
	}
	if !up.confirm(latest) {
		return nil
	}

	up.Logf("c2n: running 'plugin update tailscale.plg'")
	cmd := exec.Command("plugin", "update", "tailscale.plg")
	cmd.Stdout = up.Stdout
	cmd.Stderr = up.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed tailscale plugin update: %w", err)
	}
	return nil
}

func parseUnraidPluginVersion(plgPath string) (string, error) {
	plg, err := os.ReadFile(plgPath)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile(`<FILE Name="/boot/config/plugins/tailscale/tailscale_(\d+\.\d+\.\d+)_[a-z0-9]+.tgz">`)
	match := re.FindStringSubmatch(string(plg))
	if len(match) < 2 {
		return "", errors.New("version not found in plg file")
	}
	return match[1], nil
}

func writeFile(r io.Reader, path string, perm os.FileMode) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing file at %q: %w", path, err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, r); err != nil {
		return err
	}
	return f.Close()
}

// Var allows overriding this in tests.
var binaryPaths = func() (tailscale, tailscaled string, err error) {
	// This can be either tailscale or tailscaled.
	this, err := os.Executable()
	if err != nil {
		return "", "", err
	}
	otherName := "tailscaled"
	if filepath.Base(this) == "tailscaled" {
		otherName = "tailscale"
	}
	// Try to find the other binary in the same directory.
	other := filepath.Join(filepath.Dir(this), otherName)
	_, err = os.Stat(other)
	if os.IsNotExist(err) {
		// If it's not in the same directory, try to find it in $PATH.
		other, err = exec.LookPath(otherName)
	}
	if err != nil {
		return "", "", fmt.Errorf("cannot find %q in neither %q nor $PATH: %w", otherName, filepath.Dir(this), err)
	}
	if otherName == "tailscaled" {
		return this, other, nil
	} else {
		return other, this, nil
	}
}

func haveExecutable(name string) bool {
	path, err := exec.LookPath(name)
	return err == nil && path != ""
}

func requestedTailscaleVersion(ver, track string) (string, error) {
	if ver != "" {
		return ver, nil
	}
	return LatestTailscaleVersion(track)
}

// LatestTailscaleVersion returns the latest released version for the given
// track from pkgs.tailscale.com.
func LatestTailscaleVersion(track string) (string, error) {
	if track == CurrentTrack {
		if version.IsUnstableBuild() {
			track = UnstableTrack
		} else {
			track = StableTrack
		}
	}

	latest, err := latestPackages(track)
	if err != nil {
		return "", err
	}
	if latest.Version == "" {
		return "", fmt.Errorf("no latest version found for %q track", track)
	}
	return latest.Version, nil
}

type trackPackages struct {
	Version         string
	Tarballs        map[string]string
	TarballsVersion string
	Exes            []string
	ExesVersion     string
	MSIs            map[string]string
	MSIsVersion     string
	MacZips         map[string]string
	MacZipsVersion  string
	SPKs            map[string]map[string]string
	SPKsVersion     string
}

func latestPackages(track string) (*trackPackages, error) {
	url := fmt.Sprintf("https://pkgs.tailscale.com/%s/?mode=json&os=%s", track, runtime.GOOS)
	res, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetching latest tailscale version: %w", err)
	}
	defer res.Body.Close()
	var latest trackPackages
	if err := json.NewDecoder(res.Body).Decode(&latest); err != nil {
		return nil, fmt.Errorf("decoding JSON: %v: %w", res.Status, err)
	}
	return &latest, nil
}

func requireRoot() error {
	if os.Geteuid() == 0 {
		return nil
	}
	switch runtime.GOOS {
	case "linux":
		return errors.New("must be root; use sudo")
	case "freebsd", "openbsd":
		return errors.New("must be root; use doas")
	default:
		return errors.New("must be root")
	}
}

func isExitError(err error) bool {
	var exitErr *exec.ExitError
	return errors.As(err, &exitErr)
}
