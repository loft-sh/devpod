package rstudio

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/devpod/pkg/config"
	copypkg "github.com/loft-sh/devpod/pkg/copy"
	devpodhttp "github.com/loft-sh/devpod/pkg/http"
	"github.com/loft-sh/devpod/pkg/ide"
	"github.com/loft-sh/devpod/pkg/single"
	"github.com/loft-sh/log"
)

const (
	OpenOption        = "OPEN"
	BindAddressOption = "BIND_ADDRESS"
)

var Options = ide.Options{
	BindAddressOption: {
		Name:        BindAddressOption,
		Description: "The address to bind the server to locally. E.g. 0.0.0.0:12345",
		Default:     "",
	},
	OpenOption: {
		Name:        OpenOption,
		Description: "If DevPod should automatically open the browser",
		Default:     "true",
		Enum: []string{
			"true",
			"false",
		},
	},
}

const (
	DefaultServerPort = 8787

	downloadFolder = "/var/devpod/rstudio-server"
	dataFolder     = "/usr/local/share/devpod/rstudio-server/data"
	// rstudioConfigFolder is where RStudio expects configuration
	rstudioConfigFolder = "/etc/rstudio"
	preferencesFile     = "rstudio-prefs.json"

	defaultBinaryPathRStudioServer = "/usr/lib/rstudio-server/bin/rstudio-server"
	defaultBinaryPathRServer       = "/usr/lib/rstudio-server/bin/rserver"
)

type preferences struct {
	InitialWorkingDirectory string `json:"initial_working_directory,omitempty"` // RStudio expects snake_case
}

func NewRStudioServer(workspaceFolder string, userName string, values map[string]config.OptionValue, log log.Logger) *RStudioServer {
	return &RStudioServer{
		values:          values,
		workspaceFolder: workspaceFolder,
		userName:        userName,
		log:             log,
	}
}

type RStudioServer struct {
	values          map[string]config.OptionValue
	workspaceFolder string
	userName        string
	log             log.Logger
}

var codenameRegEx = regexp.MustCompile(`\nUBUNTU_CODENAME=(.*)\n`)

func (o *RStudioServer) Install() error {
	debPath := filepath.Join(filepath.ToSlash(downloadFolder), "rstudio-server.deb")
	// R has to be installed
	if !command.ExistsForUser("R", o.userName) {
		return fmt.Errorf("R has to be available in image to use RStudio") //nolint:all
	}

	// Skip if already installed
	if command.ExistsForUser("rstudio-server", o.userName) {
		o.log.Debug("RStudio is already installed, skipping installation")
		return nil
	}
	o.log.Info("Installing RStudio")

	err := ensureGdebi(o.log)
	if err != nil {
		return err
	}

	// Check if local file exists
	if _, err := os.Stat(debPath); os.IsNotExist(err) {
		o.log.Info("Rstudio deb not file, downloading ...")
		codename, err := getDistroCodename(o.log)
		if err != nil {
			return err
		}

		debPath, err = downloadRStudioDeb(codename, o.log)
		if err != nil {
			return err
		}
	}

	err = installDeb(debPath, o.log)
	if err != nil {
		return err
	}

	err = ensureConfigFolder(o.userName)
	if err != nil {
		return err
	}

	err = setupSingleUserMode(dataFolder, o.userName)
	if err != nil {
		return err
	}

	err = setupPreferences(o.workspaceFolder, o.userName)
	if err != nil {
		return err
	}
	o.log.Done("Successfully installed RStudio")

	return o.Start()
}

func (o *RStudioServer) Start() error {
	return single.Single("rstudio.pid", func() (*exec.Cmd, error) {
		o.log.Info("Starting RStudio...")
		runCommand := "rstudio-server start"
		args := []string{}
		if o.userName != "" {
			args = append(args, "su", o.userName, "-w", "SSH_AUTH_SOCK", "-l", "-c", runCommand)
		} else {
			args = append(args, "sh", "-l", "-c", runCommand)
		}
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = o.workspaceFolder
		return cmd, nil
	})
}

func ensureGdebi(log log.Logger) error {
	if !command.Exists("gdebi") {
		log.Info("Installing dependency gdebi-core")
		out, err := exec.Command("apt", "update").CombinedOutput()
		if err != nil {
			return fmt.Errorf("apt update: %w: %s", err, string(out))
		}

		out, err = exec.Command("apt", "-y", "install", "--no-install-recommends", "gdebi-core").CombinedOutput()
		if err != nil {
			return fmt.Errorf("install gdebi core: %w: %s", err, string(out))
		}
	}

	return nil
}

func getDistroCodename(log log.Logger) (string, error) {
	// Base distro needs to be ubuntu
	all, err := os.ReadFile("/etc/os-release")
	if err != nil {
		return "", fmt.Errorf("read /etc/os-release: %w", err)
	}
	if !bytes.Contains(all, []byte("ID=ubuntu")) {
		return "", fmt.Errorf("RStudio Server is only supported on ubuntu images, OS information is %s", string(all))
	}

	// Find ubuntu release codename
	matches := codenameRegEx.FindStringSubmatch(string(all))
	if len(matches) < 2 {
		return "", fmt.Errorf("unable to find ubuntu release codename")
	}
	ubuntuCodename := strings.Trim(matches[1], `"`)
	log.Debug("Found ubuntu codename", ubuntuCodename)

	return ubuntuCodename, nil
}

func downloadRStudioDeb(ubuntuCodename string, log log.Logger) (string, error) {
	downloadURL := getDownloadURL("stable", ubuntuCodename, runtime.GOARCH) // the agent injection already handles the cpu architecture
	log.Infof("Downloading RStudio from %s", downloadURL)

	// Download .deb
	debPath, err := download(downloadFolder, downloadURL, log)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}
	log.Done("Successfully downloaded RStudio")

	return debPath, nil
}

func getDownloadURL(version, ubuntuCodename, architecture string) string {
	return "https://rstudio.org/download/latest/" + version + "/server/" + ubuntuCodename + "/rstudio-server-latest-" + architecture + ".deb"
}

func download(targetFolder, downloadURL string, log log.Logger) (string, error) {
	err := os.MkdirAll(targetFolder, os.ModePerm)
	if err != nil {
		return "", err
	}

	targetPath := filepath.Join(filepath.ToSlash(targetFolder), "rstudio-server.deb")

	resp, err := devpodhttp.GetHTTPClient().Get(downloadURL)
	if err != nil {
		return "", fmt.Errorf("download deb: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		if resp.StatusCode == http.StatusNotFound {
			return "", fmt.Errorf("RStudio version doesn't exist: %s", downloadURL) //nolint:all
		}

		return "", fmt.Errorf("download binary returned status code %d", resp.StatusCode)
	}

	stat, err := os.Stat(targetPath)
	if err == nil && stat.Size() == resp.ContentLength {
		return targetPath, nil
	}

	file, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.Copy(file, &ide.ProgressReader{
		Reader:    resp.Body,
		TotalSize: resp.ContentLength,
		Log:       log,
	})
	if err != nil {
		return "", fmt.Errorf("download file: %w", err)
	}

	return targetPath, nil
}

func installDeb(debPath string, log log.Logger) error {
	log.Info("Installing deb")

	out, err := exec.Command("gdebi", "--non-interactive", debPath).CombinedOutput()
	if err != nil {
		return fmt.Errorf("install deb: %w: %s", err, string(out))
	}

	// The installer unfortunately automatically starts a new rstudio process without reading the configuration.
	// We're stopping that so that our process can take over later on
	out, err = exec.Command(defaultBinaryPathRStudioServer, "stop").CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop initial RStudio process: %w: %s", err, string(out))
	}

	// The RStudio deb installs into /usr/lib/rstudio-server/bin/rstudio-server by default
	// and symlinks the binaries to /usr/sbin.
	// We need to symlink to /usr/local/bin to ensure remoteUser has access later on
	err = os.Symlink(defaultBinaryPathRStudioServer, "/usr/local/bin/rstudio-server")
	if err != nil {
		return fmt.Errorf("symlink rstudio-server: %w", err)
	}
	err = os.Symlink(defaultBinaryPathRServer, "/usr/local/bin/rserver")
	if err != nil {
		return fmt.Errorf("symlink rserver: %w", err)
	}

	return nil
}

func ensureConfigFolder(userName string) error {
	err := os.MkdirAll(dataFolder, os.ModePerm)
	if err != nil {
		return err
	}

	err = copypkg.ChownR(dataFolder, userName)
	if err != nil {
		return err
	}

	return nil
}

func setupSingleUserMode(configFolder, userName string) error {
	// Check out https://docs.posit.co/ide/server-pro/rstudio-server-configuration.html for details
	dbConf := fmt.Sprintf(`provider=sqlite
directory=%s`, configFolder)
	dbConfPath := filepath.Join(configFolder, "dbconf.conf")
	err := os.WriteFile(dbConfPath, []byte(dbConf), os.ModePerm)
	if err != nil {
		return fmt.Errorf("save db conf: %w", err)
	}

	rServerConf := fmt.Sprintf(`# https://docs.posit.co/ide/server-pro/access_and_security/server_permissions.html#running-without-permissions
server-user=%s
auth-none=1
auth-minimum-user-id=0

server-data-dir=%s
database-config-file=%s/dbconf.conf
`, userName, configFolder, configFolder)
	serverConfPath := filepath.Join(rstudioConfigFolder, "rserver.conf")
	// The RStudio installer automatically creates an empty file at destConfPath, let's try to remove that first
	_ = os.Remove(serverConfPath)
	err = os.WriteFile(serverConfPath, []byte(rServerConf), os.ModePerm)
	if err != nil {
		return fmt.Errorf("save rserver conf: %w", err)
	}

	return nil
}

func setupPreferences(workspaceFolder, userName string) error {
	homeDir, err := command.GetHome(userName)
	if err != nil {
		return fmt.Errorf("get home dir")
	}
	prefsDir := filepath.Join(homeDir, ".config", "rstudio")
	err = os.MkdirAll(prefsDir, os.ModePerm)
	if err != nil {
		return err
	}
	err = copypkg.ChownR(prefsDir, userName)
	if err != nil {
		return err
	}

	prefs := preferences{InitialWorkingDirectory: workspaceFolder}
	outPrefs, err := json.Marshal(prefs)
	if err != nil {
		return err
	}

	prefsPath := filepath.Join(prefsDir, "rstudio-prefs.json")
	err = os.WriteFile(prefsPath, outPrefs, os.ModePerm)
	if err != nil {
		return fmt.Errorf("save preferences: %w", err)
	}
	err = copypkg.Chown(prefsPath, userName)
	if err != nil {
		return err
	}

	return nil
}
