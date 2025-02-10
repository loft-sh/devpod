package workspace

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/loft-sh/devpod/pkg/git"
)

var (
	workspaceIDRegEx1 = regexp.MustCompile(`[^\w\-]`)
	workspaceIDRegEx2 = regexp.MustCompile(`[^0-9a-z\-]+`)

	branchRegEx      = regexp.MustCompile(`[^a-zA-Z0-9\.\-]+`)
	prReferenceRegEx = regexp.MustCompile(git.PullRequestReference)
)

func ToID(str string) string {
	str = strings.ToLower(filepath.ToSlash(str))
	splitted := strings.Split(str, "@")
	if len(splitted) == 2 {
		// 1. Check if PR was specified
		if prReferenceRegEx.MatchString(str) {
			str = prReferenceRegEx.ReplaceAllStringFunc(splitted[1], git.GetIDForPR)
		} else {
			// 2. Check if a branch name has been specified, if so use this for the ID
			str = strings.TrimSuffix(splitted[1], ".git")
			// Check if branch name matches expected regex
			if !branchRegEx.MatchString(str) {
				str = splitted[0]
			}
		}
	} else {
		// Ensure we don't have a single trailing slash
		str = strings.TrimSuffix(str, "/")
		// 3. If not, then parse the repo name as ID
		index := strings.LastIndex(str, "/")
		if index != -1 {
			str = str[index+1:]

			// remove a potential tag / branch name
			if len(splitted) == 2 && !branchRegEx.MatchString(splitted[1]) {
				str = splitted[0]
			}

			// remove .git if there is it
			str = strings.TrimSuffix(str, ".git")
		}
	}

	str = workspaceIDRegEx2.ReplaceAllString(workspaceIDRegEx1.ReplaceAllString(str, "-"), "")
	if len(str) > 48 {
		str = str[:48]
	}

	return strings.Trim(str, "-")
}
