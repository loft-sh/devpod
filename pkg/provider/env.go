package provider

import (
	"os"
	"strings"
)

const (
	WORKSPACE_ID             = "WORKSPACE_ID"
	WORKSPACE_CONTEXT        = "WORKSPACE_CONTEXT"
	WORKSPACE_ORIGIN         = "WORKSPACE_ORIGIN"
	WORKSPACE_GIT_REPOSITORY = "WORKSPACE_GIT_REPOSITORY"
	WORKSPACE_GIT_BRANCH     = "WORKSPACE_GIT_BRANCH"
	WORKSPACE_GIT_COMMIT     = "WORKSPACE_GIT_COMMIT"
	WORKSPACE_LOCAL_FOLDER   = "WORKSPACE_LOCAL_FOLDER"
	WORKSPACE_IMAGE          = "WORKSPACE_IMAGE"
	WORKSPACE_PROVIDER       = "WORKSPACE_PROVIDER"
)

func FromEnvironment() *Workspace {
	return &Workspace{
		ID: os.Getenv(WORKSPACE_ID),
		Source: WorkspaceSource{
			GitRepository: os.Getenv(WORKSPACE_GIT_REPOSITORY),
			GitBranch:     os.Getenv(WORKSPACE_GIT_BRANCH),
			GitCommit:     os.Getenv(WORKSPACE_GIT_COMMIT),
			LocalFolder:   os.Getenv(WORKSPACE_LOCAL_FOLDER),
			Image:         os.Getenv(WORKSPACE_IMAGE),
		},
		Provider: WorkspaceProviderConfig{
			Name: os.Getenv(WORKSPACE_PROVIDER),
		},
		Context: os.Getenv(WORKSPACE_CONTEXT),
		Origin:  os.Getenv(WORKSPACE_ORIGIN),
	}
}

func ToEnvironment(workspace *Workspace) []string {
	retVars := []string{}
	if workspace.ID != "" {
		retVars = append(retVars, WORKSPACE_ID+"="+workspace.ID)
	}
	if workspace.Context != "" {
		retVars = append(retVars, WORKSPACE_CONTEXT+"="+workspace.Context)
	}
	if workspace.Origin != "" {
		retVars = append(retVars, WORKSPACE_ORIGIN+"="+workspace.Origin)
	}
	if workspace.Source.LocalFolder != "" {
		retVars = append(retVars, WORKSPACE_LOCAL_FOLDER+"="+workspace.Source.LocalFolder)
	}
	if workspace.Source.GitRepository != "" {
		retVars = append(retVars, WORKSPACE_GIT_REPOSITORY+"="+workspace.Source.GitRepository)
	}
	if workspace.Source.GitBranch != "" {
		retVars = append(retVars, WORKSPACE_GIT_BRANCH+"="+workspace.Source.GitBranch)
	}
	if workspace.Source.GitCommit != "" {
		retVars = append(retVars, WORKSPACE_GIT_COMMIT+"="+workspace.Source.GitCommit)
	}
	if workspace.Source.Image != "" {
		retVars = append(retVars, WORKSPACE_IMAGE+"="+workspace.Source.Image)
	}
	if workspace.Provider.Name != "" {
		retVars = append(retVars, WORKSPACE_PROVIDER+"="+workspace.Provider.Name)
	}
	for optionName, optionValue := range workspace.Provider.Options {
		retVars = append(retVars, strings.ToUpper(optionName)+"="+optionValue)
	}

	return retVars
}
