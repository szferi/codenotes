package engine

import (
	"errors"
	tpl "html/template"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

type Engine struct {
	layout *tpl.Template
}

func New(layoutFS fs.FS, layoutPatterns ...string) (*Engine, error) {
	e := &Engine{}
	var err error
	e.layout, err = e.ParseFS(nil, layoutFS, layoutPatterns...)
	if err != nil {
		return nil, err
	}
	return e, nil
}

func (e *Engine) Layout() *tpl.Template {
	return e.layout
}

func (e *Engine) ParseFS(t *tpl.Template, fsys fs.FS, patterns ...string) (*tpl.Template, error) {
	err := fs.WalkDir(fsys, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		matched := false
		for _, pattern := range patterns {
			base := filepath.Base(path)
			matched, err = filepath.Match(pattern, base)
			if err != nil {
				return err
			}
			if matched {
				break
			}
		}
		// if there was no match, we skip the file
		if !matched {
			return nil
		}
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		s := string(b)
		// First template becomes return value if not already defined,
		// and we use that one for subsequent New calls to associate
		// all the templates together. Also, if this file has the same name
		// as t, this file becomes the contents of t, so
		//  t, err := New(name).Funcs(xxx).ParseFiles(name)
		// works. Otherwise we create a new template associated with t.
		var tmpl *tpl.Template
		if t == nil {
			t = tpl.New(path)
		}
		if path == t.Name() {
			tmpl = t
		} else {
			tmpl = t.New(path)
		}
		_, err = tmpl.Parse(s)
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New("empty template")
	}
	return t, nil
}

func (e *Engine) ExecuteTemplate(w io.Writer, templatePath string, data any) error {
	if e.layout == nil {
		return errors.New("empty layout")
	}
	var t *tpl.Template
	var err error
	t, err = e.layout.Clone()
	if err != nil {
		return err
	}
	b, err := os.ReadFile(templatePath)
	if err != nil {
		return err
	}
	s := string(b)
	_, err = t.New(templatePath).Parse(s)
	if err != nil {
		return err
	}
	return t.ExecuteTemplate(w, templatePath, data)
}
