package util

import (
	"bytes"
	"io/fs"
	"text/template"
)

func RenderTemplateFS(fsys fs.FS, path string, data any) (string, error) {
	b, err := fs.ReadFile(fsys, path)
	if err != nil {
		return "", err
	}

	tpl, err := template.New(path).Parse(string(b))
	if err != nil {
		return "", err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}
