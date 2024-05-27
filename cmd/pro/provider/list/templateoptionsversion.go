package list

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	managementv1 "github.com/loft-sh/api/v4/pkg/apis/management/v1"
	storagev1 "github.com/loft-sh/api/v4/pkg/apis/storage/v1"

	"github.com/loft-sh/devpod/cmd/pro/flags"
	"github.com/loft-sh/devpod/pkg/loft"
	"github.com/loft-sh/devpod/pkg/loft/client"
	"github.com/loft-sh/log"

	"github.com/blang/semver"
	"github.com/spf13/cobra"
)

// TemplateOptionsVersionCmd holds the cmd flags
type TemplateOptionsVersionCmd struct {
	*flags.GlobalFlags

	log log.Logger
}

// NewTemplateOptionsVersionCmd creates a new command
func NewTemplateOptionsVersionCmd(globalFlags *flags.GlobalFlags) *cobra.Command {
	cmd := &TemplateOptionsVersionCmd{
		GlobalFlags: globalFlags,
		log:         log.GetInstance(),
	}
	c := &cobra.Command{
		Use:   "templateoptionsversion",
		Short: "Lists template options for a specific version for the DevPod provider",
		Args:  cobra.NoArgs,
		RunE: func(cobraCmd *cobra.Command, args []string) error {
			return cmd.Run(cobraCmd.Context())
		},
	}

	return c
}

func (cmd *TemplateOptionsVersionCmd) Run(ctx context.Context) error {
	projectName := os.Getenv(loft.ProjectEnv)
	if projectName == "" {
		return fmt.Errorf("%s environment variable is empty", loft.ProjectEnv)
	}
	templateName := os.Getenv(loft.TemplateOptionEnv)
	if templateName == "" {
		return fmt.Errorf("%s environment variable is empty", loft.TemplateOptionEnv)
	}
	templateVersion := os.Getenv(loft.TemplateVersionOptionEnv)
	if templateName == "" {
		return fmt.Errorf("%s environment variable is empty", loft.TemplateVersionOptionEnv)
	}

	baseClient, err := client.InitClientFromPath(ctx, cmd.Config)
	if err != nil {
		return err
	}

	managementClient, err := baseClient.Management()
	if err != nil {
		return err
	}

	// check template
	template, err := FindTemplate(ctx, managementClient, projectName, templateName)
	if err != nil {
		return err
	}

	// get parameters
	parameters, err := GetTemplateParameters(template, templateVersion)
	if err != nil {
		return err
	}

	// print to stdout
	return printOptions(&OptionsFormat{Options: parametersToOptions(parameters)})
}

func GetTemplateParameters(template *managementv1.DevPodWorkspaceTemplate, templateVersion string) ([]storagev1.AppParameter, error) {
	if templateVersion == "latest" {
		templateVersion = ""
	}

	if templateVersion == "" {
		latestVersion := GetLatestVersion(template)
		if latestVersion == nil {
			return nil, fmt.Errorf("couldn't find any version in template")
		}

		return latestVersion.(*storagev1.DevPodWorkspaceTemplateVersion).Parameters, nil
	}

	_, latestMatched, err := GetLatestMatchedVersion(template, templateVersion)
	if err != nil {
		return nil, err
	} else if latestMatched == nil {
		return nil, fmt.Errorf("couldn't find any matching version to %s", templateVersion)
	}

	return latestMatched.(*storagev1.DevPodWorkspaceTemplateVersion).Parameters, nil
}

type matchedVersion struct {
	Object  storagev1.VersionAccessor
	Version semver.Version
}

func GetLatestVersion(versions storagev1.VersionsAccessor) storagev1.VersionAccessor {
	// find the latest version
	var latestVersion *matchedVersion
	for _, version := range versions.GetVersions() {
		parsedVersion, err := semver.Parse(strings.TrimPrefix(version.GetVersion(), "v"))
		if err != nil {
			continue
		}

		// latest available version
		if latestVersion == nil || latestVersion.Version.LT(parsedVersion) {
			latestVersion = &matchedVersion{
				Object:  version,
				Version: parsedVersion,
			}
		}
	}
	if latestVersion == nil {
		return nil
	}

	return latestVersion.Object
}

func GetLatestMatchedVersion(versions storagev1.VersionsAccessor, versionPattern string) (latestVersion storagev1.VersionAccessor, latestMatchedVersion storagev1.VersionAccessor, err error) {
	// parse version
	splittedVersion := strings.Split(strings.ToLower(strings.TrimPrefix(versionPattern, "v")), ".")
	if len(splittedVersion) != 3 {
		return nil, nil, fmt.Errorf("couldn't parse version %s, expected version in format: 0.0.0", versionPattern)
	}

	// find latest version that matches our defined version
	var latestVersionObj *matchedVersion
	var latestMatchedVersionObj *matchedVersion
	for _, version := range versions.GetVersions() {
		parsedVersion, err := semver.Parse(strings.TrimPrefix(version.GetVersion(), "v"))
		if err != nil {
			continue
		}

		// does the version match our restrictions?
		if (splittedVersion[0] == "x" || splittedVersion[0] == "X" || strconv.FormatUint(parsedVersion.Major, 10) == splittedVersion[0]) &&
			(splittedVersion[1] == "x" || splittedVersion[1] == "X" || strconv.FormatUint(parsedVersion.Minor, 10) == splittedVersion[1]) &&
			(splittedVersion[2] == "x" || splittedVersion[2] == "X" || strconv.FormatUint(parsedVersion.Patch, 10) == splittedVersion[2]) {
			if latestMatchedVersionObj == nil || latestMatchedVersionObj.Version.LT(parsedVersion) {
				latestMatchedVersionObj = &matchedVersion{
					Object:  version,
					Version: parsedVersion,
				}
			}
		}

		// latest available version
		if latestVersionObj == nil || latestVersionObj.Version.LT(parsedVersion) {
			latestVersionObj = &matchedVersion{
				Object:  version,
				Version: parsedVersion,
			}
		}
	}

	if latestVersionObj != nil {
		latestVersion = latestVersionObj.Object
	}
	if latestMatchedVersionObj != nil {
		latestMatchedVersion = latestMatchedVersionObj.Object
	}

	return latestVersion, latestMatchedVersion, nil
}
