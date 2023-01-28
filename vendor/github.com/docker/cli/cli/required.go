package cli

import (
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NoArgs validates args and returns an error if there are any args
func NoArgs(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return nil
	}

	if cmd.HasSubCommands() {
		return errors.Errorf("\n" + strings.TrimRight(cmd.UsageString(), "\n"))
	}

	return errors.Errorf(
		"%q accepts no arguments.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
		cmd.CommandPath(),
		cmd.CommandPath(),
		cmd.UseLine(),
		cmd.Short,
	)
}

// RequiresMinArgs returns an error if there is not at least min args
func RequiresMinArgs(min int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) >= min {
			return nil
		}
		return errors.Errorf(
			"%q requires at least %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			min,
			pluralize("argument", min),
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

// RequiresMaxArgs returns an error if there is not at most max args
func RequiresMaxArgs(max int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) <= max {
			return nil
		}
		return errors.Errorf(
			"%q requires at most %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			max,
			pluralize("argument", max),
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

// RequiresRangeArgs returns an error if there is not at least min args and at most max args
func RequiresRangeArgs(min int, max int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) >= min && len(args) <= max {
			return nil
		}
		return errors.Errorf(
			"%q requires at least %d and at most %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			min,
			max,
			pluralize("argument", max),
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

// ExactArgs returns an error if there is not the exact number of args
func ExactArgs(number int) cobra.PositionalArgs {
	return func(cmd *cobra.Command, args []string) error {
		if len(args) == number {
			return nil
		}
		return errors.Errorf(
			"%q requires exactly %d %s.\nSee '%s --help'.\n\nUsage:  %s\n\n%s",
			cmd.CommandPath(),
			number,
			pluralize("argument", number),
			cmd.CommandPath(),
			cmd.UseLine(),
			cmd.Short,
		)
	}
}

//nolint: unparam
func pluralize(word string, number int) string {
	if number == 1 {
		return word
	}
	return word + "s"
}
