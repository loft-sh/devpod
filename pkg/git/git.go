package git

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"github.com/loft-sh/devpod/pkg/command"
	"github.com/loft-sh/log"
	"github.com/sirupsen/logrus"
)

const (
	CommitDelimiter      string = "@sha256:"
	PullRequestReference string = "pull/([0-9]+)/head"
	SubPathDelimiter     string = "@subpath:"
)

// WARN: Make sure this matches the regex in /desktop/src/views/Workspaces/CreateWorkspace/CreateWorkspaceInput.tsx!
var (
	// Updated regex pattern to support SSH-style Git URLs
	repoBaseRegEx    = `((?:(?:https?|git|ssh|file):\/\/)?\/?(?:[^@\/\n]+@)?(?:[^:\/\n]+)(?:[:\/][^\/\n]+)+(?:\.git)?)`
	branchRegEx      = regexp.MustCompile(`^` + repoBaseRegEx + `@([a-zA-Z0-9\./\-\_]+)$`)
	commitRegEx      = regexp.MustCompile(`^` + repoBaseRegEx + regexp.QuoteMeta(CommitDelimiter) + `([a-zA-Z0-9]+)$`)
	prReferenceRegEx = regexp.MustCompile(`^` + repoBaseRegEx + `@(` + PullRequestReference + `)$`)
	subPathRegEx     = regexp.MustCompile(`^` + repoBaseRegEx + regexp.QuoteMeta(SubPathDelimiter) + `([a-zA-Z0-9\./\-\_]+)$`)
)

func NormalizeRepository(str string) (string, string, string, string, string) {
	if !strings.HasPrefix(str, "ssh://") &&
		!strings.HasPrefix(str, "git@") &&
		!strings.HasPrefix(str, "http://") &&
		!strings.HasPrefix(str, "https://") &&
		!strings.HasPrefix(str, "file://") {
		str = "https://" + str
	}

	// resolve pull request reference
	prReference := ""
	if match := prReferenceRegEx.FindStringSubmatch(str); match != nil {
		str = match[1]
		prReference = match[2]

		return str, prReference, "", "", ""
	}

	// resolve subpath
	subpath := ""
	if match := subPathRegEx.FindStringSubmatch(str); match != nil {
		str = match[1]
		subpath = strings.TrimSuffix(match[2], "/")
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

	return str, prReference, branch, commit, subpath
}

func CommandContext(ctx context.Context, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "GIT_TERMINAL_PROMPT=0")
	cmd.Env = append(cmd.Env, "GIT_SSH_COMMAND=ssh -oBatchMode=yes -oStrictHostKeyChecking=no")
	return cmd
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

func GetIDForPR(ref string) string {
	regex := regexp.MustCompile(PullRequestReference)
	return regex.ReplaceAllString(ref, "pr${1}")
}

type GitInfo struct {
	Repository string
	Branch     string
	Commit     string
	PR         string
	SubPath    string
}

func NewGitInfo(repository, branch, commit, pr, subpath string) *GitInfo {
	return &GitInfo{
		Repository: repository,
		Branch:     branch,
		Commit:     commit,
		PR:         pr,
		SubPath:    subpath,
	}
}

func NormalizeRepositoryGitInfo(str string) *GitInfo {
	repository, pr, branch, commit, subpath := NormalizeRepository(str)
	return NewGitInfo(repository, branch, commit, pr, subpath)
}

func CloneRepository(ctx context.Context, gitInfo *GitInfo, targetDir string, helper string, cloner Cloner, log log.Logger) error {
	return CloneRepositoryWithEnv(ctx, gitInfo, []string{}, targetDir, helper, cloner, log)
}

func CloneRepositoryWithEnv(ctx context.Context, gitInfo *GitInfo, extraEnv []string, targetDir string, helper string, cloner Cloner, log log.Logger) error {
	if cloner == nil {
		cloner = NewCloner(FullCloneStrategy)
	}

	extraArgs := []string{}
	if helper != "" {
		extraArgs = append(extraArgs, "--config", "credential.helper="+helper)
	}

	if gitInfo.Branch != "" {
		extraArgs = append(extraArgs, "--branch", gitInfo.Branch)
	}

	if err := cloner.Clone(ctx, gitInfo.Repository, targetDir, extraArgs, extraEnv, log); err != nil {
		return err
	}

	if gitInfo.PR != "" {
		return checkoutPR(ctx, gitInfo, targetDir, log)
	}

	if gitInfo.Commit != "" {
		return checkoutCommit(ctx, gitInfo, targetDir, log)
	}

	return nil
}

func checkoutPR(ctx context.Context, gitInfo *GitInfo, targetDir string, log log.Logger) error {
	log.Debugf("Fetching pull request : %s", gitInfo.PR)

	prBranch := GetBranchNameForPR(gitInfo.PR)

	// Try to fetch the pull request by
	// checking out the reference GitHub set up for it. Afterwards, switch to it.
	// See [this doc](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/reviewing-changes-in-pull-requests/checking-out-pull-requests-locally#modifying-an-inactive-pull-request-locally)
	// Command args: `git fetch origin pull/996/head:PR996`
	fetchArgs := []string{"fetch", "origin", gitInfo.PR + ":" + prBranch}
	fetchCmd := CommandContext(ctx, fetchArgs...)
	fetchCmd.Dir = targetDir
	if err := fetchCmd.Run(); err != nil {
		return fmt.Errorf("fetch pull request reference: %w", err)
	}

	// git switch PR996
	switchArgs := []string{"switch", prBranch}
	switchCmd := CommandContext(ctx, switchArgs...)
	switchCmd.Dir = targetDir
	if err := switchCmd.Run(); err != nil {
		return fmt.Errorf("switch to branch: %w", err)
	}

	return nil
}

func checkoutCommit(ctx context.Context, gitInfo *GitInfo, targetDir string, log log.Logger) error {
	stdout := log.Writer(logrus.InfoLevel, false)
	stderr := log.Writer(logrus.ErrorLevel, false)
	defer stdout.Close()
	defer stderr.Close()

	args := []string{"reset", "--hard", gitInfo.Commit}
	gitCommand := CommandContext(ctx, args...)
	gitCommand.Dir = targetDir
	gitCommand.Stdout = stdout
	gitCommand.Stderr = stderr
	if err := gitCommand.Run(); err != nil {
		return fmt.Errorf("reset head to commit: %w", err)
	}

	return nil
}
