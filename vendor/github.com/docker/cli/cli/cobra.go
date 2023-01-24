package cli

import (
	"fmt"
	"os"
	"strings"

	pluginmanager "github.com/docker/cli/cli-plugins/manager"
	"github.com/docker/cli/cli/command"
	cliconfig "github.com/docker/cli/cli/config"
	cliflags "github.com/docker/cli/cli/flags"
	"github.com/moby/term"
	"github.com/morikuni/aec"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// setupCommonRootCommand contains the setup common to
// SetupRootCommand and SetupPluginRootCommand.
func setupCommonRootCommand(rootCmd *cobra.Command) (*cliflags.ClientOptions, *pflag.FlagSet, *cobra.Command) {
	opts := cliflags.NewClientOptions()
	flags := rootCmd.Flags()

	flags.StringVar(&opts.ConfigDir, "config", cliconfig.Dir(), "Location of client config files")
	opts.Common.InstallFlags(flags)

	cobra.AddTemplateFunc("add", func(a, b int) int { return a + b })
	cobra.AddTemplateFunc("hasSubCommands", hasSubCommands)
	cobra.AddTemplateFunc("hasManagementSubCommands", hasManagementSubCommands)
	cobra.AddTemplateFunc("hasInvalidPlugins", hasInvalidPlugins)
	cobra.AddTemplateFunc("operationSubCommands", operationSubCommands)
	cobra.AddTemplateFunc("managementSubCommands", managementSubCommands)
	cobra.AddTemplateFunc("invalidPlugins", invalidPlugins)
	cobra.AddTemplateFunc("wrappedFlagUsages", wrappedFlagUsages)
	cobra.AddTemplateFunc("vendorAndVersion", vendorAndVersion)
	cobra.AddTemplateFunc("invalidPluginReason", invalidPluginReason)
	cobra.AddTemplateFunc("isPlugin", isPlugin)
	cobra.AddTemplateFunc("isExperimental", isExperimental)
	cobra.AddTemplateFunc("hasAdditionalHelp", hasAdditionalHelp)
	cobra.AddTemplateFunc("additionalHelp", additionalHelp)
	cobra.AddTemplateFunc("decoratedName", decoratedName)

	rootCmd.SetUsageTemplate(usageTemplate)
	rootCmd.SetHelpTemplate(helpTemplate)
	rootCmd.SetFlagErrorFunc(FlagErrorFunc)
	rootCmd.SetHelpCommand(helpCommand)

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	rootCmd.PersistentFlags().MarkShorthandDeprecated("help", "please use --help")
	rootCmd.PersistentFlags().Lookup("help").Hidden = true

	rootCmd.Annotations = map[string]string{"additionalHelp": "To get more help with docker, check out our guides at https://docs.docker.com/go/guides/"}

	return opts, flags, helpCommand
}

// SetupRootCommand sets default usage, help, and error handling for the
// root command.
func SetupRootCommand(rootCmd *cobra.Command) (*cliflags.ClientOptions, *pflag.FlagSet, *cobra.Command) {
	opts, flags, helpCmd := setupCommonRootCommand(rootCmd)

	rootCmd.SetVersionTemplate("Docker version {{.Version}}\n")

	return opts, flags, helpCmd
}

// SetupPluginRootCommand sets default usage, help and error handling for a plugin root command.
func SetupPluginRootCommand(rootCmd *cobra.Command) (*cliflags.ClientOptions, *pflag.FlagSet) {
	opts, flags, _ := setupCommonRootCommand(rootCmd)
	return opts, flags
}

// FlagErrorFunc prints an error message which matches the format of the
// docker/cli/cli error messages
func FlagErrorFunc(cmd *cobra.Command, err error) error {
	if err == nil {
		return nil
	}

	usage := ""
	if cmd.HasSubCommands() {
		usage = "\n\n" + cmd.UsageString()
	}
	return StatusError{
		Status:     fmt.Sprintf("%s\nSee '%s --help'.%s", err, cmd.CommandPath(), usage),
		StatusCode: 125,
	}
}

// TopLevelCommand encapsulates a top-level cobra command (either
// docker CLI or a plugin) and global flag handling logic necessary
// for plugins.
type TopLevelCommand struct {
	cmd       *cobra.Command
	dockerCli *command.DockerCli
	opts      *cliflags.ClientOptions
	flags     *pflag.FlagSet
	args      []string
}

// NewTopLevelCommand returns a new TopLevelCommand object
func NewTopLevelCommand(cmd *cobra.Command, dockerCli *command.DockerCli, opts *cliflags.ClientOptions, flags *pflag.FlagSet) *TopLevelCommand {
	return &TopLevelCommand{cmd, dockerCli, opts, flags, os.Args[1:]}
}

// SetArgs sets the args (default os.Args[:1] used to invoke the command
func (tcmd *TopLevelCommand) SetArgs(args []string) {
	tcmd.args = args
	tcmd.cmd.SetArgs(args)
}

// SetFlag sets a flag in the local flag set of the top-level command
func (tcmd *TopLevelCommand) SetFlag(name, value string) {
	tcmd.cmd.Flags().Set(name, value)
}

// HandleGlobalFlags takes care of parsing global flags defined on the
// command, it returns the underlying cobra command and the args it
// will be called with (or an error).
//
// On success the caller is responsible for calling Initialize()
// before calling `Execute` on the returned command.
func (tcmd *TopLevelCommand) HandleGlobalFlags() (*cobra.Command, []string, error) {
	cmd := tcmd.cmd

	// We manually parse the global arguments and find the
	// subcommand in order to properly deal with plugins. We rely
	// on the root command never having any non-flag arguments. We
	// create our own FlagSet so that we can configure it
	// (e.g. `SetInterspersed` below) in an idempotent way.
	flags := pflag.NewFlagSet(cmd.Name(), pflag.ContinueOnError)

	// We need !interspersed to ensure we stop at the first
	// potential command instead of accumulating it into
	// flags.Args() and then continuing on and finding other
	// arguments which we try and treat as globals (when they are
	// actually arguments to the subcommand).
	flags.SetInterspersed(false)

	// We need the single parse to see both sets of flags.
	flags.AddFlagSet(cmd.Flags())
	flags.AddFlagSet(cmd.PersistentFlags())
	// Now parse the global flags, up to (but not including) the
	// first command. The result will be that all the remaining
	// arguments are in `flags.Args()`.
	if err := flags.Parse(tcmd.args); err != nil {
		// Our FlagErrorFunc uses the cli, make sure it is initialized
		if err := tcmd.Initialize(); err != nil {
			return nil, nil, err
		}
		return nil, nil, cmd.FlagErrorFunc()(cmd, err)
	}

	return cmd, flags.Args(), nil
}

// Initialize finalises global option parsing and initializes the docker client.
func (tcmd *TopLevelCommand) Initialize(ops ...command.InitializeOpt) error {
	tcmd.opts.Common.SetDefaultOptions(tcmd.flags)
	return tcmd.dockerCli.Initialize(tcmd.opts, ops...)
}

// VisitAll will traverse all commands from the root.
// This is different from the VisitAll of cobra.Command where only parents
// are checked.
func VisitAll(root *cobra.Command, fn func(*cobra.Command)) {
	for _, cmd := range root.Commands() {
		VisitAll(cmd, fn)
	}
	fn(root)
}

// DisableFlagsInUseLine sets the DisableFlagsInUseLine flag on all
// commands within the tree rooted at cmd.
func DisableFlagsInUseLine(cmd *cobra.Command) {
	VisitAll(cmd, func(ccmd *cobra.Command) {
		// do not add a `[flags]` to the end of the usage line.
		ccmd.DisableFlagsInUseLine = true
	})
}

var helpCommand = &cobra.Command{
	Use:               "help [command]",
	Short:             "Help about the command",
	PersistentPreRun:  func(cmd *cobra.Command, args []string) {},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {},
	RunE: func(c *cobra.Command, args []string) error {
		cmd, args, e := c.Root().Find(args)
		if cmd == nil || e != nil || len(args) > 0 {
			return errors.Errorf("unknown help topic: %v", strings.Join(args, " "))
		}
		helpFunc := cmd.HelpFunc()
		helpFunc(cmd, args)
		return nil
	},
}

func isExperimental(cmd *cobra.Command) bool {
	if _, ok := cmd.Annotations["experimentalCLI"]; ok {
		return true
	}
	var experimental bool
	cmd.VisitParents(func(cmd *cobra.Command) {
		if _, ok := cmd.Annotations["experimentalCLI"]; ok {
			experimental = true
		}
	})
	return experimental
}

func additionalHelp(cmd *cobra.Command) string {
	if msg, ok := cmd.Annotations["additionalHelp"]; ok {
		out := cmd.OutOrStderr()
		if _, isTerminal := term.GetFdInfo(out); !isTerminal {
			return msg
		}
		style := aec.EmptyBuilder.Bold().ANSI
		return style.Apply(msg)
	}
	return ""
}

func hasAdditionalHelp(cmd *cobra.Command) bool {
	return additionalHelp(cmd) != ""
}

func isPlugin(cmd *cobra.Command) bool {
	return cmd.Annotations[pluginmanager.CommandAnnotationPlugin] == "true"
}

func hasSubCommands(cmd *cobra.Command) bool {
	return len(operationSubCommands(cmd)) > 0
}

func hasManagementSubCommands(cmd *cobra.Command) bool {
	return len(managementSubCommands(cmd)) > 0
}

func hasInvalidPlugins(cmd *cobra.Command) bool {
	return len(invalidPlugins(cmd)) > 0
}

func operationSubCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isPlugin(sub) {
			continue
		}
		if sub.IsAvailableCommand() && !sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func wrappedFlagUsages(cmd *cobra.Command) string {
	width := 80
	if ws, err := term.GetWinsize(0); err == nil {
		width = int(ws.Width)
	}
	return cmd.Flags().FlagUsagesWrapped(width - 1)
}

func decoratedName(cmd *cobra.Command) string {
	decoration := " "
	if isPlugin(cmd) {
		decoration = "*"
	}
	return cmd.Name() + decoration
}

func vendorAndVersion(cmd *cobra.Command) string {
	if vendor, ok := cmd.Annotations[pluginmanager.CommandAnnotationPluginVendor]; ok && isPlugin(cmd) {
		version := ""
		if v, ok := cmd.Annotations[pluginmanager.CommandAnnotationPluginVersion]; ok && v != "" {
			version = ", " + v
		}
		return fmt.Sprintf("(%s%s)", vendor, version)
	}
	return ""
}

func managementSubCommands(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if isPlugin(sub) {
			if invalidPluginReason(sub) == "" {
				cmds = append(cmds, sub)
			}
			continue
		}
		if sub.IsAvailableCommand() && sub.HasSubCommands() {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func invalidPlugins(cmd *cobra.Command) []*cobra.Command {
	cmds := []*cobra.Command{}
	for _, sub := range cmd.Commands() {
		if !isPlugin(sub) {
			continue
		}
		if invalidPluginReason(sub) != "" {
			cmds = append(cmds, sub)
		}
	}
	return cmds
}

func invalidPluginReason(cmd *cobra.Command) string {
	return cmd.Annotations[pluginmanager.CommandAnnotationPluginInvalid]
}

var usageTemplate = `Usage:

{{- if not .HasSubCommands}}  {{.UseLine}}{{end}}
{{- if .HasSubCommands}}  {{ .CommandPath}}{{- if .HasAvailableFlags}} [OPTIONS]{{end}} COMMAND{{end}}

{{if ne .Long ""}}{{ .Long | trim }}{{ else }}{{ .Short | trim }}{{end}}
{{- if isExperimental .}}

EXPERIMENTAL:
  {{.CommandPath}} is an experimental feature.
  Experimental features provide early access to product functionality. These
  features may change between releases without warning, or can be removed from a
  future release. Learn more about experimental features in our documentation:
  https://docs.docker.com/go/experimental/

{{- end}}
{{- if gt .Aliases 0}}

Aliases:
  {{.NameAndAliases}}

{{- end}}
{{- if .HasExample}}

Examples:
{{ .Example }}

{{- end}}
{{- if .HasAvailableFlags}}

Options:
{{ wrappedFlagUsages . | trimRightSpace}}

{{- end}}
{{- if hasManagementSubCommands . }}

Management Commands:

{{- range managementSubCommands . }}
  {{rpad (decoratedName .) (add .NamePadding 1)}}{{.Short}}{{ if isPlugin .}} {{vendorAndVersion .}}{{ end}}
{{- end}}

{{- end}}
{{- if hasSubCommands .}}

Commands:

{{- range operationSubCommands . }}
  {{rpad .Name .NamePadding }} {{.Short}}
{{- end}}
{{- end}}

{{- if hasInvalidPlugins . }}

Invalid Plugins:

{{- range invalidPlugins . }}
  {{rpad .Name .NamePadding }} {{invalidPluginReason .}}
{{- end}}

{{- end}}

{{- if .HasSubCommands }}

Run '{{.CommandPath}} COMMAND --help' for more information on a command.
{{- end}}
{{- if hasAdditionalHelp .}}

{{ additionalHelp . }}

{{- end}}
`

var helpTemplate = `
{{if or .Runnable .HasSubCommands}}{{.UsageString}}{{end}}`
