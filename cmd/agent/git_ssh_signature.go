package agent

import (
	"errors"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/gitsshsigning"
	"github.com/loft-sh/log"
	"github.com/spf13/cobra"
)

type GitSSHSignatureCmd struct {
	*flags.GlobalFlags

	CertPath   string
	Namespace  string
	BufferFile string
	Command    string
}

// NewGitSSHSignatureCmd creates new git-ssh-signature command
// This agent command can be used as git ssh program by setting
//
//	> git config --global gpg.ssh.program "devpod agent git-ssh-signature"
//
// Git by default uses ssh-keygen for signing commits with ssh. This CLI command is a drop-in
// replacement for ssh-keygen and hence needs to support ssh-keygen interface that git uses.
//
//	custom-ssh-signature-handler -Y sign -n git -f /Users/johndoe/.ssh/my-key.pub /tmp/.git_signing_buffer_tmp4Euk6d
func NewGitSSHSignatureCmd(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GitSSHSignatureCmd{
		GlobalFlags: flags,
	}

	gitSshSignatureCmd := &cobra.Command{
		Use: "git-ssh-signature",
		RunE: func(_ *cobra.Command, args []string) error {
			logger := log.GetInstance()

			if len(args) < 1 {
				logger.Fatalf("Buffer file is required")
			}

			// Check if the required -Y sign flags are present
			if cmd.Command != "sign" {
				return errors.New("must include '-Y sign' arguments")
			}

			// The last argument is the buffer file
			cmd.BufferFile = args[len(args)-1]

			return gitsshsigning.HandleGitSSHProgramCall(
				cmd.CertPath, cmd.Namespace, cmd.BufferFile, logger)
		},
	}

	gitSshSignatureCmd.Flags().StringVarP(&cmd.CertPath, "file", "f", "", "Path to the private key")
	gitSshSignatureCmd.Flags().StringVarP(&cmd.Namespace, "namespace", "n", "", "Namespace")
	gitSshSignatureCmd.Flags().StringVarP(&cmd.Command, "command", "Y", "sign", "Command - should be 'sign'")

	return gitSshSignatureCmd
}
