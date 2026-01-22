// Package template provides HTML template utilities for the Quark framework.
// It wraps html/template with common patterns for web development, including
// template reloading for development, embedded filesystem support, and a rich
// set of built-in template functions.
//
// Basic usage:
//
//	// Create template engine
//	config := template.DefaultConfig()
//	config.Dir = "views"
//	config.Reload = true  // Enable reload in development
//	engine, err := template.New(config)
//
//	// Render in handler
//	app.GET("/", func(c *quark.Context) error {
//	    data := quark.M{
//	        "title": "Home",
//	        "user":  user,
//	    }
//	    return engine.HTML(c, 200, "home", data)
//	})
//
// With embedded templates:
//
//	//go:embed templates/*
//	var templatesFS embed.FS
//
//	config := template.DefaultConfig()
//	config.Extension = ".html"
//	engine, err := template.NewFromFS(templatesFS, config)
//
// Available template functions:
//   - safeHTML, safeURL, safeAttr, safeJS, safeCSS: Safe output functions
//   - lower, upper, title, trim, replace, contains, etc.: String manipulation
//   - eq, ne: Comparison operators
//   - add, sub, mul, div, mod: Arithmetic operators
//   - default: Default value if empty
//   - classIf: Conditional CSS class
//   - plural: Pluralization helper
//   - truncate: Text truncation
//   - dict, list: Data structure helpers
package template

import (
	"bytes"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/AchrafSoltani/quark"
)

// Engine is a template engine that manages HTML templates.
type Engine struct {
	templates *template.Template
	funcMap   template.FuncMap
	dir       string
	ext       string
	reload    bool
	mu        sync.RWMutex
}

// Config holds template engine configuration.
type Config struct {
	// Dir is the directory containing template files.
	Dir string

	// Extension is the template file extension (default: ".html").
	Extension string

	// Reload enables template reloading on each request (for development).
	Reload bool

	// FuncMap is the template function map.
	FuncMap template.FuncMap

	// Layouts is a list of layout template paths relative to Dir.
	Layouts []string
}

// DefaultConfig returns the default template configuration.
func DefaultConfig() Config {
	return Config{
		Dir:       "templates",
		Extension: ".html",
		Reload:    false,
		FuncMap:   make(template.FuncMap),
		Layouts:   []string{},
	}
}

// New creates a new template engine.
func New(config Config) (*Engine, error) {
	if config.Dir == "" {
		config.Dir = "templates"
	}
	if config.Extension == "" {
		config.Extension = ".html"
	}
	if config.FuncMap == nil {
		config.FuncMap = make(template.FuncMap)
	}

	// Add default functions
	addDefaultFuncs(config.FuncMap)

	engine := &Engine{
		funcMap: config.FuncMap,
		dir:     config.Dir,
		ext:     config.Extension,
		reload:  config.Reload,
	}

	if err := engine.load(); err != nil {
		return nil, err
	}

	return engine, nil
}

// NewFromFS creates a template engine from an embedded filesystem.
func NewFromFS(fsys fs.FS, config Config) (*Engine, error) {
	if config.Extension == "" {
		config.Extension = ".html"
	}
	if config.FuncMap == nil {
		config.FuncMap = make(template.FuncMap)
	}

	addDefaultFuncs(config.FuncMap)

	tmpl := template.New("").Funcs(config.FuncMap)

	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, config.Extension) {
			return nil
		}

		content, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}

		name := strings.TrimSuffix(path, config.Extension)
		_, err = tmpl.New(name).Parse(string(content))
		return err
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return &Engine{
		templates: tmpl,
		funcMap:   config.FuncMap,
		dir:       config.Dir,
		ext:       config.Extension,
		reload:    false, // No reload for embedded FS
	}, nil
}

// load loads all templates from the directory.
func (e *Engine) load() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	tmpl := template.New("").Funcs(e.funcMap)

	err := filepath.Walk(e.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, e.ext) {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Use relative path as template name (without extension)
		relPath, _ := filepath.Rel(e.dir, path)
		name := strings.TrimSuffix(relPath, e.ext)
		name = filepath.ToSlash(name) // Normalize to forward slashes

		_, err = tmpl.New(name).Parse(string(content))
		return err
	})

	if err != nil {
		return fmt.Errorf("failed to load templates: %w", err)
	}

	e.templates = tmpl
	return nil
}

// Reload reloads all templates.
func (e *Engine) Reload() error {
	return e.load()
}

// Render renders a template to a writer.
func (e *Engine) Render(w io.Writer, name string, data interface{}) error {
	if e.reload {
		if err := e.load(); err != nil {
			return err
		}
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	tmpl := e.templates.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("template not found: %s", name)
	}

	return tmpl.Execute(w, data)
}

// RenderString renders a template to a string.
func (e *Engine) RenderString(name string, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := e.Render(&buf, name, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

// ExecuteTemplate renders a template with the given name.
func (e *Engine) ExecuteTemplate(w io.Writer, name string, data interface{}) error {
	return e.Render(w, name, data)
}

// AddFunc adds a template function.
func (e *Engine) AddFunc(name string, fn interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.funcMap[name] = fn
}

// HTML renders a template and sends the result as an HTML response.
func (e *Engine) HTML(c *quark.Context, code int, name string, data interface{}) error {
	var buf bytes.Buffer
	if err := e.Render(&buf, name, data); err != nil {
		return quark.WrapError(http.StatusInternalServerError, "template rendering failed", err)
	}

	c.SetHeader("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteHeader(code)
	_, err := c.Writer.Write(buf.Bytes())
	return err
}

// addDefaultFuncs adds default template functions.
func addDefaultFuncs(fm template.FuncMap) {
	// Safe HTML output
	fm["safeHTML"] = func(s string) template.HTML {
		return template.HTML(s)
	}

	// Safe URL output
	fm["safeURL"] = func(s string) template.URL {
		return template.URL(s)
	}

	// Safe attribute output
	fm["safeAttr"] = func(s string) template.HTMLAttr {
		return template.HTMLAttr(s)
	}

	// Safe JavaScript output
	fm["safeJS"] = func(s string) template.JS {
		return template.JS(s)
	}

	// Safe CSS output
	fm["safeCSS"] = func(s string) template.CSS {
		return template.CSS(s)
	}

	// String manipulation
	fm["lower"] = strings.ToLower
	fm["upper"] = strings.ToUpper
	fm["title"] = strings.Title
	fm["trim"] = strings.TrimSpace
	fm["replace"] = strings.ReplaceAll
	fm["contains"] = strings.Contains
	fm["hasPrefix"] = strings.HasPrefix
	fm["hasSuffix"] = strings.HasSuffix
	fm["split"] = strings.Split
	fm["join"] = strings.Join

	// Comparison
	fm["eq"] = func(a, b interface{}) bool { return a == b }
	fm["ne"] = func(a, b interface{}) bool { return a != b }

	// Arithmetic
	fm["add"] = func(a, b int) int { return a + b }
	fm["sub"] = func(a, b int) int { return a - b }
	fm["mul"] = func(a, b int) int { return a * b }
	fm["div"] = func(a, b int) int { return a / b }
	fm["mod"] = func(a, b int) int { return a % b }

	// Default value
	fm["default"] = func(def, val interface{}) interface{} {
		if val == nil || val == "" || val == 0 || val == false {
			return def
		}
		return val
	}

	// Conditional class
	fm["classIf"] = func(condition bool, class string) string {
		if condition {
			return class
		}
		return ""
	}

	// Pluralize
	fm["plural"] = func(count int, singular, plural string) string {
		if count == 1 {
			return singular
		}
		return plural
	}

	// Sequence generation
	fm["seq"] = func(n int) []int {
		result := make([]int, n)
		for i := range result {
			result[i] = i
		}
		return result
	}

	// Range (for pagination, etc.)
	fm["rangeN"] = func(start, end int) []int {
		if end < start {
			return []int{}
		}
		result := make([]int, end-start)
		for i := range result {
			result[i] = start + i
		}
		return result
	}

	// Truncate text
	fm["truncate"] = func(s string, n int) string {
		if len(s) <= n {
			return s
		}
		return s[:n] + "..."
	}

	// Dict for passing multiple values to templates
	fm["dict"] = func(values ...interface{}) (map[string]interface{}, error) {
		if len(values)%2 != 0 {
			return nil, fmt.Errorf("dict expects an even number of arguments")
		}
		dict := make(map[string]interface{}, len(values)/2)
		for i := 0; i < len(values); i += 2 {
			key, ok := values[i].(string)
			if !ok {
				return nil, fmt.Errorf("dict keys must be strings")
			}
			dict[key] = values[i+1]
		}
		return dict, nil
	}

	// List for creating slices
	fm["list"] = func(values ...interface{}) []interface{} {
		return values
	}
}

// Renderer interface for Quark integration.
type Renderer interface {
	Render(io.Writer, string, interface{}) error
}

// Ensure Engine implements Renderer
var _ Renderer = (*Engine)(nil)
