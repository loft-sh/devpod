package git

import (
	"context"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
)

const (
	CommitDelimiter      string = "@sha256:"
	PullRequestReference string = "pull/([0-9]+)/head"
)

var (
	branchRegEx      = regexp.MustCompile(`^([^@]*(?:git@)?[^@/]+/[^@/]+/[^@/]+)@([a-zA-Z0-9\./\-\_]+)$`)
	commitRegEx      = regexp.MustCompile(`^([^@]*(?:git@)?[^@/]+/[^@]+)` + regexp.QuoteMeta(CommitDelimiter) + `([a-zA-Z0-9]+)$`)
	prReferenceRegEx = regexp.MustCompile(`^([^@]*(?:git@)?[^@/]+/[^@]+)@(` + PullRequestReference + `)$`)
)

func CommandContext(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -oBatchMode=yes -oStrictHostKeyChecking=no")
	return cmd
}

func NormalizeRepository(str string) (string, string, string, string) {
	if !strings.HasPrefix(str, "ssh://") && !strings.HasPrefix(str, "git@") && !strings.HasPrefix(str, "http://") && !strings.HasPrefix(str, "https://") {
		str = "https://" + str
	}

	// resolve pull request reference
	prReference := ""
	if match := prReferenceRegEx.FindStringSubmatch(str); match != nil {
		str = match[1]
		prReference = match[2]

		return str, prReference, "", ""
	}

	// resolve branch
	branch := ""
	if match := branchRegEx.FindStringSubmatch(str); match != nil {
		str = match[1]
		branch = match[2]
	}

	// resolve commit hash
	commit := ""
	if match := commitRegEx.FindStringSubmatch(str); match != nil {
		str = match[1]
		commit = match[2]
	}

	return str, prReference, branch, commit
}

func PingRepository(str string) bool {
	if !command.Exists("git") {
		return false
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := CommandContext(timeoutCtx, "ls-remote", "--quiet", str).CombinedOutput()
	return err == nil
}

func GetBranchNameForPR(ref string) string {
	regex := regexp.MustCompile(PullRequestReference)
	return regex.ReplaceAllString(ref, "PR${1}")
}
