package git

import (
	"context"
	"io"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
)

const (
	CommitDelimiter      string = "@sha256:"
	PullRequestReference string = "pull/([0-9]+)/head"
)

var (
	branchRegEx      = regexp.MustCompile(`^([^@]*(?:git@)?[^@/]+/[^@/]+/?[^@/]+)@([a-zA-Z0-9\./\-\_]+)$`)
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

type GitInfo struct {
	Repository string
	Branch     string
	Commit     string
	PR         string
}

func NewGitInfo(repository, branch, commit, pr string) *GitInfo {
	return &GitInfo{
		Repository: repository,
		Branch:     branch,
		Commit:     commit,
		PR:         pr,
	}
}

func NormalizeRepositoryGitInfo(str string) *GitInfo {
	repository, pr, branch, commit := NormalizeRepository(str)
	return NewGitInfo(repository, branch, commit, pr)
}

func CloneRepository(ctx context.Context, gitInfo *GitInfo, targetDir string, helper string, bare bool, writer io.Writer, log log.Logger) error {
	args := []string{"clone"}
	if bare && gitInfo.Commit == "" {
		args = append(args, "--bare", "--depth=1")
	}
	if helper != "" {
		args = append(args, "--config", "credential.helper="+helper)
	}
	if gitInfo.Branch != "" {
		args = append(args, "--branch", gitInfo.Branch)
	}
	args = append(args, gitInfo.Repository, targetDir)
	gitCommand := CommandContext(ctx, args...)
	gitCommand.Stdout = writer
	gitCommand.Stderr = writer
	err := gitCommand.Run()
	if err != nil {
		return errors.Wrap(err, "error cloning repository")
	}

	if gitInfo.PR != "" {
		log.Debugf("Fetching pull request : %s", gitInfo.PR)

		prBranch := GetBranchNameForPR(gitInfo.PR)

		// git fetch origin pull/996/head:PR996
		fetchArgs := []string{"fetch", "origin", gitInfo.PR + ":" + prBranch}
		fetchCmd := CommandContext(ctx, fetchArgs...)
		fetchCmd.Dir = targetDir
		err = fetchCmd.Run()
		if err != nil {
			return errors.Wrap(err, "error fetching pull request reference")
		}

		// git switch PR996
		switchArgs := []string{"switch", prBranch}
		switchCmd := CommandContext(ctx, switchArgs...)
		switchCmd.Dir = targetDir
		err = switchCmd.Run()
		if err != nil {
			return errors.Wrap(err, "error switching to the branch")
		}
	} else if gitInfo.Commit != "" {
		args := []string{"reset", "--hard", gitInfo.Commit}
		gitCommand := CommandContext(ctx, args...)
		gitCommand.Dir = targetDir
		gitCommand.Stdout = writer
		gitCommand.Stderr = writer
		err := gitCommand.Run()
		if err != nil {
			return errors.Wrap(err, "error resetting head to commit")
		}
	}
	return nil
}
