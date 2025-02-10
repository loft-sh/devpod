package telemetry

import (
	"encoding/json"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/loft-sh/analytics-client/client"
	devpodclient "github.com/loft-sh/devpod/pkg/client"
	"github.com/loft-sh/devpod/pkg/config"
	"github.com/loft-sh/devpod/pkg/version"
	"github.com/loft-sh/log"
	"github.com/moby/term"
	"github.com/spf13/cobra"
)

type ErrorSeverityType string

const (
	WarningSeverity ErrorSeverityType = "warning"
	ErrorSeverity   ErrorSeverityType = "error"
	FatalSeverity   ErrorSeverityType = "fatal"
	PanicSeverity   ErrorSeverityType = "panic"
)

const UIEnvVar = "DEVPOD_UI"

var UIEventsExceptions []string = []string{
	"devpod list",
	"devpod status",
	"devpod provider list",
	"devpod pro list",
	"devpod pro check-health",
	"devpod pro check-update",
	"devpod ide list",
	"devpod ide use",
	"devpod provider use",
	"devpod version",
	"devpod context options",
}

// skip everything in pro mode
var CollectorCLI CLICollector = &noopCollector{}

type CLICollector interface {
	RecordCLI(err error)
	SetClient(client devpodclient.BaseWorkspaceClient)

	// Flush makes sure all events are sent to the backend
	Flush()
}

// StartCLI starts collecting events and sending them to the backend from the CLI
func StartCLI(devPodConfig *config.Config, cmd *cobra.Command) {
	telemetryOpt := devPodConfig.ContextOption(config.ContextOptionTelemetry)
	if telemetryOpt == "false" || version.GetVersion() == version.DevVersion ||
		os.Getenv("DEVPOD_DISABLE_TELEMETRY") == "true" {
		return
	}

	// create a new default collector
	collector, err := newCLICollector(cmd)
	if err != nil {
		// Log the problem but don't fail - use disabled Collector instead
		log.Default.WithPrefix("telemetry").Infof("%s", err.Error())
	} else {
		CollectorCLI = collector
	}
}

func newCLICollector(cmd *cobra.Command) (*cliCollector, error) {
	defaultCollector := &cliCollector{
		analyticsClient: client.NewClient(),
		log:             log.Default.WithPrefix("telemetry"),
		cmd:             cmd,
	}

	return defaultCollector, nil
}

type cliCollector struct {
	analyticsClient client.Client
	cmd             *cobra.Command
	client          devpodclient.BaseWorkspaceClient

	log log.Logger
}

func (d *cliCollector) SetClient(client devpodclient.BaseWorkspaceClient) {
	d.client = client
}

func (d *cliCollector) Flush() {
	d.analyticsClient.Flush()
}

func (d *cliCollector) RecordCLI(err error) {
	if d.cmd == nil {
		d.log.Debug("no command found, skipping")
		return
	}
	cmd := d.cmd.CommandPath()
	isUI := false
	if os.Getenv(UIEnvVar) == "true" {
		isUI = true
	}
	// Ignore certain commands triggered by DevPod Desktop
	if isUI {
		for _, exception := range UIEventsExceptions {
			if cmd == exception {
				return
			}
		}
	}

	isCI := false
	if !isUI {
		isCI = isCIEnvironment()
	}

	isInteractive := false
	if !isUI {
		isInteractive = isInteractiveShell()
	}

	timezone, _ := time.Now().Zone()
	eventProperties := map[string]interface{}{
		"command":        cmd,
		"version":        version.GetVersion(),
		"desktop":        isUI,
		"is_ci":          isCI,
		"is_interactive": isInteractive,
	}
	if d.client != nil {
		eventProperties["provider"] = d.client.Provider()

		if d.client.WorkspaceConfig() != nil {
			eventProperties["source_type"] = d.client.WorkspaceConfig().Source.Type()
			eventProperties["ide"] = d.client.WorkspaceConfig().IDE.Name
		}
	}
	userProperties := map[string]interface{}{
		"os_name":  runtime.GOOS,
		"os_arch":  runtime.GOARCH,
		"timezone": timezone,
	}
	if err != nil {
		eventProperties["error"] = err.Error()
	}

	// Check if we're on the runner
	isPro := false
	wd, wdErr := os.Getwd()
	if wdErr == nil {
		if strings.HasPrefix(wd, "/var/lib/loft/devpod") {
			isPro = true
		}
	}
	eventType := "devpod_cli"
	if isPro {
		eventType = "devpod_cli_runner"
	}

	// build the event and record
	eventPropertiesRaw, _ := json.Marshal(eventProperties)
	userPropertiesRaw, _ := json.Marshal(userProperties)
	d.analyticsClient.RecordEvent(client.Event{
		"event": {
			"type":       eventType,
			"machine_id": GetMachineID(),
			"properties": string(eventPropertiesRaw),
			"timestamp":  time.Now().Unix(),
		},
		"user": {
			"machine_id": GetMachineID(),
			"properties": string(userPropertiesRaw),
			"timestamp":  time.Now().Unix(),
		},
	})
}

// isCIEnvironment looks up a couple of well-known CI env vars
func isCIEnvironment() bool {
	ciIndicators := []string{
		"CI",                     // Generic CI variable
		"TRAVIS",                 // Travis CI
		"GITHUB_ACTIONS",         // GitHub Actions
		"GITLAB_CI",              // GitLab CI
		"CIRCLECI",               // CircleCI
		"TEAMCITY_VERSION",       // TeamCity
		"BITBUCKET_BUILD_NUMBER", // Bitbucket
	}

	for _, key := range ciIndicators {
		if _, exists := os.LookupEnv(key); exists {
			return true
		}
	}
	return false
}

// isInteractiveShell checks if the current shell is in interactive mode or not.
// Can be combined with `isCi` to narrow down usage
func isInteractiveShell() bool {
	return term.IsTerminal(os.Stdin.Fd())
}
