package wave

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateLoader resolves and renders wave templates.
type TemplateLoader struct {
	projectRoot string
}

func NewTemplateLoader(projectRoot string) *TemplateLoader {
	return &TemplateLoader{projectRoot: projectRoot}
}

func (l *TemplateLoader) Resolve(templateName string) (string, error) {
	name := strings.TrimSpace(templateName)
	if name == "" {
		return "", fmt.Errorf("template is required")
	}

	if filepath.IsAbs(name) {
		if _, err := os.Stat(name); err != nil {
			return "", fmt.Errorf("template not found: %s", name)
		}
		return name, nil
	}

	candidate := name
	if filepath.Ext(candidate) == "" {
		candidate += ".tmpl"
	}

	full := filepath.Join(l.projectRoot, "internal", "actions", "templates", candidate)
	if _, err := os.Stat(full); err != nil {
		return "", fmt.Errorf("template not found: %s", full)
	}
	return full, nil
}

func (l *TemplateLoader) Render(templateName string, data any) (string, error) {
	path, err := l.Resolve(templateName)
	if err != nil {
		return "", err
	}

	tpl, err := template.New(filepath.Base(path)).ParseFiles(path)
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", path, err)
	}

	var out bytes.Buffer
	if err := tpl.Execute(&out, data); err != nil {
		return "", fmt.Errorf("render template %s: %w", path, err)
	}
	return out.String(), nil
}
