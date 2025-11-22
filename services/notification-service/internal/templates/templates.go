package templates

import (
	"bytes"
	"embed"
	"fmt"
	"path/filepath"
	"strings"
	"text/template"

	"shared/pkg/logger"
)

//go:embed *.tmpl
var templatesFS embed.FS

// TemplateEngine manages notification templates
type TemplateEngine struct {
	templates map[string]*template.Template
	logger    *logger.Logger
}

// NewTemplateEngine creates a new template engine
func NewTemplateEngine(logger *logger.Logger, templateDir string) (*TemplateEngine, error) {
	engine := &TemplateEngine{
		templates: make(map[string]*template.Template),
		logger:    logger,
	}

	// Load templates from filesystem if directory is provided
	if templateDir != "" {
		if err := engine.loadFromDir(templateDir); err != nil {
			return nil, fmt.Errorf("failed to load templates from dir: %w", err)
		}
	} else {
		// Load embedded templates
		if err := engine.loadEmbedded(); err != nil {
			return nil, fmt.Errorf("failed to load embedded templates: %w", err)
		}
	}

	logger.Info().
		Int("count", len(engine.templates)).
		Msg("Template engine initialized")

	return engine, nil
}

// loadEmbedded loads templates from embedded filesystem
func (te *TemplateEngine) loadEmbedded() error {
	entries, err := templatesFS.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read embedded templates: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tmpl") {
			continue
		}

		content, err := templatesFS.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read template %s: %w", entry.Name(), err)
		}

		name := strings.TrimSuffix(entry.Name(), ".tmpl")
		tmpl, err := template.New(name).Funcs(te.funcMap()).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		te.templates[name] = tmpl
		te.logger.Debug().Str("template", name).Msg("Loaded embedded template")
	}

	return nil
}

// loadFromDir loads templates from a directory
func (te *TemplateEngine) loadFromDir(dir string) error {
	pattern := filepath.Join(dir, "*.tmpl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob templates: %w", err)
	}

	for _, file := range files {
		name := strings.TrimSuffix(filepath.Base(file), ".tmpl")
		tmpl, err := template.New(name).Funcs(te.funcMap()).ParseFiles(file)
		if err != nil {
			return fmt.Errorf("failed to parse template %s: %w", name, err)
		}

		te.templates[name] = tmpl
		te.logger.Debug().Str("template", name).Str("file", file).Msg("Loaded template from file")
	}

	return nil
}

// Render renders a template with the given data
func (te *TemplateEngine) Render(name string, data interface{}) (string, error) {
	tmpl, ok := te.templates[name]
	if !ok {
		return "", fmt.Errorf("template %s not found", name)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute template %s: %w", name, err)
	}

	return buf.String(), nil
}

// RenderWithFallback renders a template with fallback to default template
func (te *TemplateEngine) RenderWithFallback(name, fallback string, data interface{}) (string, error) {
	result, err := te.Render(name, data)
	if err != nil {
		te.logger.Warn().
			Err(err).
			Str("template", name).
			Str("fallback", fallback).
			Msg("Template not found, using fallback")

		return te.Render(fallback, data)
	}
	return result, nil
}

// HasTemplate checks if a template exists
func (te *TemplateEngine) HasTemplate(name string) bool {
	_, ok := te.templates[name]
	return ok
}

// ListTemplates returns all template names
func (te *TemplateEngine) ListTemplates() []string {
	names := make([]string, 0, len(te.templates))
	for name := range te.templates {
		names = append(names, name)
	}
	return names
}

// funcMap returns template functions
func (te *TemplateEngine) funcMap() template.FuncMap {
	return template.FuncMap{
		"formatFloat": func(f float64) string {
			return fmt.Sprintf("%.2f", f)
		},
		"formatPercent": func(f float64) string {
			return fmt.Sprintf("%.2f%%", f*100)
		},
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,
		"trim":  strings.TrimSpace,
	}
}

// Reload reloads all templates (useful for development)
func (te *TemplateEngine) Reload(templateDir string) error {
	te.templates = make(map[string]*template.Template)

	if templateDir != "" {
		return te.loadFromDir(templateDir)
	}

	return te.loadEmbedded()
}
