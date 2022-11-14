# Simple Extensions of the Go's Template System

The HTML template system available in the Go's standard library is powerful but missing some functionaly to use it in web applications as it is.
The main issue is that while it supports template association so but it lacks the more powerful template inheritence like Django's template engine.
This limits the ways how you can organize and reuse templates. 

One possible way to overcome this issue is to explicitly separate templates which related to a common layout of a page (like base, header, footer etc.) from the templates which defines a given page (eg. index.html, about.html etc). 

Other minor technical issue is that there is no parser function which walks though an entire directory tree in a filesystem defined by `fs.FS` interface.

In the following I build up a simple template engine which overcomes these issues.

## Better parseFS

While the `html/template` module has a `ParseFS` function it returns error if the input filesystem includes directory and does not walk though the entire tree. Looking the
source code of this function and other helper in the `go/src/text/template/helper.go` 
we can borrow some idea and create a simple derivation that extends the capabilites of 
the current `ParseFS`.

```go
func ParseFS(t *tpl.Template, fsys fs.FS, patterns ...string) (*tpl.Template, error) {
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
		if !matched {
			return nil
		}
		b, err := fs.ReadFile(fsys, path)
		if err != nil {
			return err
		}
		s := string(b)
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
```

The main idea is that we walk though the entire directory tree using `fs.WalkDir`
and parse template files that match the defined glob patterns ignoring the directories. 
All the templates in the
directory tree associated together, the idea which comes from the `parseFiles` private function 
in the mentioned `helper.go` source code.

This `ParseFS(nil, os.DirFS("templates/layout"), "*.html")` call is able to parse for example the following directory structure ignoring the `base.txt` file:

```text
templates/layout
├── base
│  ├── base.html
│  └── base.txt
├── footer.html
└── header.html
```

where the `base.html` can have the following schemantic structure:

```text
{{- define "base" }}
Base
{{- block "header" . }}{{ end }}
{{- block "content" . }}{{ end }}
{{- block "footer" . }}{{ end }}
{{ end }}
```

The issues is that if we include to the layout directory tree the templates that defines
the `content` block then multiple file will define this block which results random
output during the execution of the template. That leads us to second extension to
the standard template system

# Execution of page templates

We require that every page template have the following schemantic structure:

```text
{{- define "content" }}
Home Content
{{ end }}
{{ template "base" }}
```

The important part is the final `{{ template "base" }}` line. It tells the parser
what root templates need to be rendered here. The `base` block should exist in the
layout templates.

The basic ide of the exacution is that we create a clone of the layout Template structure
and add associate the defined path to the cloned structure and execute the path template.

The following function implements this idea:

```go
func ExecuteTemplate(layout *tpl.Template, w io.Writer, templatePath string, data any) error {
	if layout == nil {
		return errors.New("empty layout")
	}
	var t *tpl.Template
	var err error
	t, err = layout.Clone()
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
```
## Conclusion

With a simple extensions of the Go's standard template system we can get 
a template engine that can be used in web application in a similar ways as other web frameworks
template systems (eg. Django). 

The `github.com/szferi/codenotes/go-template-engine/` includes an `engine` module which packs
the above ideas into a module.

