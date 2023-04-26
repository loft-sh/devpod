package command

import (
	"github.com/alessio/shellescape"
)

func Quote(args []string) string {
	if len(args) == 0 {
		return ""
	} else if len(args) == 1 {
		return args[0]
	}

	return shellescape.QuoteCommand(args)
}
