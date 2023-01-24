package template

import (
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"strings"
)

func WriteFiles(folder string, files map[string]string) error {
	for file, content := range files {
		err := os.WriteFile(filepath.Join(folder, file), []byte(content), 0666)
		if err != nil {
			return err
		}
	}

	return nil
}

func FillTemplate(template string, vars interface{}) (string, error) {
	t, err := htmltemplate.New("gotmpl").Parse(template)
	if err != nil {
		return "", err
	}

	var buf strings.Builder
	err = t.Execute(&buf, vars)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
