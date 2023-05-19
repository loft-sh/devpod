package scripts

import (
	_ "embed"

	"github.com/loft-sh/devpod/pkg/template"
)

//go:embed wrapper.sh
var scriptWrapper string

func WrapScript(script string) (string, error) {
	// generate script
	t, err := template.FillTemplate(scriptWrapper, map[string]string{
		"Script": script,
	})
	if err != nil {
		return "", err
	}

	return t, nil
}
