package agent

import (
	"fmt"
	"os/user"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/gitsshsigning"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

type GitSSHSignatureHelperCmd struct {
	*flags.GlobalFlags

	CertPath string
}

func NewGitSSHSignatureHelperCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GitSSHSignatureCmd{
		GlobalFlags: flags,
	}

	gitSshSignatureHelperCmd := &cobra.Command{
		Use:   "git-ssh-signature-helper [signing-key-path]",
		Short: "used to inject git ssh signature helper",
		RunE: func(_ *cobra.Command, args []string) error {
			usr, err := user.Current()
			if err != nil {
				return err
			}

			if len(args) < 1 {
				return fmt.Errorf("gitSigningKey argument is required")
			}
			cmd.CertPath = args[0]

			log := log.GetInstance()
			err = gitsshsigning.ConfigureHelper(usr.Username, cmd.CertPath, log)
			if err != nil {
				return err
			}

			return nil
		},
	}

	return gitSshSignatureHelperCmd
}
