package template

import (
	"os"
	"path/filepath"
	"strings"
	"text/template"
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

func FillTemplate(templateString string, vars interface{}) (string, error) {
	t, err := template.New("gotmpl").Parse(templateString)
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
