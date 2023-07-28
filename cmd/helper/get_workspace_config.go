package helper

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/loft-sh/devpod/cmd/flags"
	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/log"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

type GetWorkspaceConfigCommand struct {
	*flags.GlobalFlags

	timeout  time.Duration
	maxDepth int
}
type GetWorkspaceConfigCommandResult struct {
	IsImage         bool     `json:"isImage"`
	IsGitRepository bool     `json:"isGitRepository"`
	IsLocal         bool     `json:"isLocal"`
	ConfigPaths     []string `json:"configPaths"`
}

// NewGetWorkspaceConfigCommand creates a new command
func NewGetWorkspaceConfigCommand(flags *flags.GlobalFlags) *cobra.Command {
	cmd := &GetWorkspaceConfigCommand{
		GlobalFlags: flags,
	}
	shellCmd := &cobra.Command{
		Use:   "get-workspace-config",
		Short: "Retrieves a workspace config",
		RunE: func(_ *cobra.Command, args []string) error {
			if cmd.maxDepth < 0 {
				log.Default.Debugf("--max-depth was %d, setting to 0", cmd.maxDepth)
				cmd.maxDepth = 0
			}

			return cmd.Run(context.Background(), args)
		},
	}

	shellCmd.Flags().DurationVar(&cmd.timeout, "timeout", 10*time.Second, "Timeout for the command, 10 seconds by default")
	shellCmd.Flags().IntVar(&cmd.maxDepth, "max-depth", 3, "Maximum depth to search for devcontainer files")

	return shellCmd
}

func (cmd *GetWorkspaceConfigCommand) Run(ctx context.Context, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("workspace source is missing")
	}
	rawSource := args[0]

	level := log.Default.GetLevel()
	if cmd.GlobalFlags.Debug {
		level = logrus.DebugLevel
	}
	logger := log.NewStdoutLogger(os.Stdin, nil, nil, level)
	logger.Debugf("Resolving devcontainer config for source: %s", rawSource)

	ctx, cancel := context.WithTimeout(context.Background(), cmd.timeout)
	defer cancel()

	done := make(chan *GetWorkspaceConfigCommandResult, 1)
	errChan := make(chan error, 1)

	tmpDir, err := os.MkdirTemp("", "devpod")
	if err != nil {
		return err
	}
	go func() {
		result, err := findDevcontainerFiles(ctx, rawSource, tmpDir, cmd.maxDepth, logger)
		if err != nil {
			errChan <- err
			return
		}
		done <- result
	}()

	select {
	case err := <-errChan:
		_ = os.RemoveAll(tmpDir)
		return errors.WithMessage(err, "unable to find devcontainer files")
	case <-ctx.Done():
		_ = os.RemoveAll(tmpDir)
		return errors.WithMessage(ctx.Err(), "timeout while searching for devcontainer files")
	case result := <-done:
		out, err := json.Marshal(result)
		if err != nil {
			return err
		}
		log.Default.Done(string(out))
	}
	defer close(done)

	return nil
}

func findDevcontainerFiles(ctx context.Context, rawSource, tmpDirPath string, maxDepth int, log log.Logger) (*GetWorkspaceConfigCommandResult, error) {
	result := &GetWorkspaceConfigCommandResult{}

	// local path
	isLocalPath, _ := file.IsLocalDir(rawSource)
	if isLocalPath {
		log.Debug("Local directory detected")
		result.IsLocal = true
		initialDepth := strings.Count(rawSource, string(filepath.Separator))
		err := filepath.WalkDir(rawSource, func(path string, info fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			depth := strings.Count(path, string(filepath.Separator)) - initialDepth
			if info.IsDir() && depth > maxDepth {
				return filepath.SkipDir
			}

			if isDevcontainerFilename(path) {
				relPath, err := filepath.Rel(rawSource, path)
				if err != nil {
					log.Warnf("Unable to get relative path for %s: %s", path, err.Error())
					return nil
				}
				result.ConfigPaths = append(result.ConfigPaths, relPath)
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	// git repo
	gitRepository, gitBranch, gitCommit := git.NormalizeRepository(rawSource)
	if strings.HasSuffix(rawSource, ".git") || git.PingRepository(gitRepository) {
		log.Debug("Git repository detected")
		result.IsGitRepository = true

		log.Debugf("Cloning git repository into %s", tmpDirPath)
		// git clone --bare --depth=1 $REPO
		cloneArgs := []string{"clone"}
		if gitCommit == "" {
			cloneArgs = append(cloneArgs, "--bare", "--depth=1")
		}
		cloneArgs = append(cloneArgs, gitRepository, tmpDirPath)
		if gitBranch != "" {
			cloneArgs = append(cloneArgs, "--branch", gitBranch)
		}
		err := git.CommandContext(ctx, cloneArgs...).Run()
		if err != nil {
			return nil, err
		}
		log.Debug("Done cloning git repository")

		if gitCommit != "" {
			log.Debugf("Resetting HEAD to %s", gitCommit)
			// git reset --hard $COMMIT_SHA
			resetArgs := []string{"reset", "--hard", gitCommit}
			resetCmd := git.CommandContext(ctx, resetArgs...)
			resetCmd.Dir = tmpDirPath
			err = resetCmd.Run()
			if err != nil {
				return nil, err
			}
			log.Debugf("HEAD is now at %s", gitCommit)
		}

		log.Debug("Listing git file tree")
		ref := "HEAD"
		// checkout on branch if available
		if gitBranch != "" {
			ref = gitBranch
		}
		// git ls-tree -r --full-name --name-only $REF
		lsArgs := []string{"ls-tree", "-r", "--full-name", "--name-only", ref}
		lsCmd := git.CommandContext(ctx, lsArgs...)
		lsCmd.Dir = tmpDirPath
		stdout, err := lsCmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		err = lsCmd.Start()
		if err != nil {
			return nil, err
		}

		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			path := scanner.Text()
			depth := strings.Count(path, string(filepath.Separator))
			if depth > maxDepth {
				continue
			}
			if isDevcontainerFilename(path) {
				result.ConfigPaths = append(result.ConfigPaths, path)
			}
		}

		err = lsCmd.Wait()
		if err != nil {
			return nil, err
		}

		return result, nil
	}

	// container image
	_, err := image.GetImage(rawSource)
	if err == nil {
		log.Debug("Container image detected")
		result.IsImage = true
		// Doesn't matter, we just want to know if it's an image
		// not going to poke around in the image fs
		return result, nil
	}

	return result, nil
}

func isDevcontainerFilename(path string) bool {
	return filepath.Base(path) == ".devcontainer.json" || filepath.Base(path) == "devcontainer.json"
}
