package devcontainer

import (
	"bufio"
	"context"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/loft-sh/devpod/pkg/file"
	"github.com/loft-sh/devpod/pkg/git"
	"github.com/loft-sh/devpod/pkg/image"
	"github.com/loft-sh/log"
)

type GetWorkspaceConfigResult struct {
	IsImage         bool     `json:"isImage"`
	IsGitRepository bool     `json:"isGitRepository"`
	IsLocal         bool     `json:"isLocal"`
	ConfigPaths     []string `json:"configPaths"`
}

func FindDevcontainerFiles(ctx context.Context, rawSource, tmpDirPath string, maxDepth int, strictHostKeyChecking bool, log log.Logger) (*GetWorkspaceConfigResult, error) {
	// local path
	isLocalPath, _ := file.IsLocalDir(rawSource)
	if isLocalPath {
		return FindFilesInLocalDir(rawSource, maxDepth, log)
	}

	// git repo
	gitRepository, gitPRReference, gitBranch, gitCommit, gitSubDir := git.NormalizeRepository(rawSource)
	if strings.HasSuffix(rawSource, ".git") || git.PingRepository(gitRepository, git.GetDefaultExtraEnv(strictHostKeyChecking)) {
		log.Debug("Git repository detected")
		return FindFilesInGitRepo(ctx, gitRepository, gitPRReference, gitBranch, gitCommit, gitSubDir, tmpDirPath, strictHostKeyChecking, maxDepth, log)
	}

	result := &GetWorkspaceConfigResult{ConfigPaths: []string{}}

	// container image
	_, err := image.GetImage(ctx, rawSource)
	if err == nil {
		log.Debug("Container image detected")
		result.IsImage = true
		// Doesn't matter, we just want to know if it's an image
		// not going to poke around in the image fs
		return result, nil
	}

	return result, nil
}

func FindFilesInLocalDir(rawSource string, maxDepth int, log log.Logger) (*GetWorkspaceConfigResult, error) {
	log.Debug("Local directory detected")
	result := &GetWorkspaceConfigResult{ConfigPaths: []string{}}
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

func FindFilesInGitRepo(ctx context.Context, gitRepository, gitPRReference, gitBranch, gitCommit, gitSubDir, tmpDirPath string, strictHostKeyChecking bool, maxDepth int, log log.Logger) (*GetWorkspaceConfigResult, error) {
	result := &GetWorkspaceConfigResult{
		ConfigPaths:     []string{},
		IsGitRepository: true,
	}

	gitInfo := git.NewGitInfo(gitRepository, gitBranch, gitCommit, gitPRReference, gitSubDir)
	log.Debugf("Cloning git repository into %s", tmpDirPath)
	err := git.CloneRepository(ctx, gitInfo, tmpDirPath, "", strictHostKeyChecking, log, git.WithCloneStrategy(git.BareCloneStrategy))
	if err != nil {
		return nil, err
	}

	log.Debug("Listing git file tree")
	ref := "HEAD"
	// checkout on branch if available
	if gitBranch != "" {
		ref = gitBranch
	}
	// git ls-tree -r --full-name --name-only $REF
	lsArgs := []string{"ls-tree", "-r", "--full-name", "--name-only", ref}
	lsCmd := git.CommandContext(ctx, git.GetDefaultExtraEnv(strictHostKeyChecking), lsArgs...)
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

func isDevcontainerFilename(path string) bool {
	return filepath.Base(path) == ".devcontainer.json" || filepath.Base(path) == "devcontainer.json"
}
