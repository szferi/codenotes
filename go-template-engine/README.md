+++
summary: A simple extension of Go's standard template system that can be used in web applications similarly, e.g. Django. 
is_pinned: False
is_published: True
published_at: 2022-11-14
+++

# Simple Extensions of the Go's Template System

The HTML template system available in Go's standard library is powerful but needs some functionality to use in web applications as it is.
The main issue is that while it supports template association, it needs more powerful template inheritance, like Django's template engine.
This limits the ways how you can organize and reuse templates.

One possible way to overcome this issue is to explicitly separate templates related to a common layout of a page (like base, header, footer, etc.) from the templates which define a given page (e.g. index.html, about.html, etc.).

Another minor technical issue is that there is no parser function that walks through an entire directory tree in a filesystem defined by the `fs.FS` interface.

In the following, I build up a simple template engine that overcomes these issues.

## Better parseFS

While the `html/template` module has a `ParseFS` function, it returns an error if the input filesystem includes a directory and does not walk though the entire tree. Looking at the
source code of this function and other helpers in the `go/src/text/template/helper.go`
we can borrow some ideas and create a simple derivation that extends the capabilities of
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

The main idea is that we walk through the entire directory tree using `fs.WalkDir`
and parse template files that match the defined glob patterns ignoring the directories.
All the templates in the
directory tree are associated, the idea which comes from the `parseFiles` private function
in the mentioned `helper.go` source code.

This `ParseFS(nil, os.DirFS("templates/layout"), "*.html")` call can parse for example, the following directory structure ignoring the `base.txt` file:

```text
templates/layout
├── base
│  ├── base.html
│  └── base.txt
├── footer.html
└── header.html
```

where the `base.html` can have the following schematic structure:

```text
{{- define "base" }}
Base
{{- block "header" . }}{{ end }}
{{- block "content" . }}{{ end }}
{{- block "footer" . }}{{ end }}
{{ end }}
```

The issue is that if we include the layout directory tree in the templates that defines
the `content` block, then multiple files will define this block which results in random
output during the execution of the template. That leads us to the second extension to
the standard template system

## Execution of page templates

We require that every page template have the following schematic structure:

```text
{{- define "content" }}
Home Content
{{ end }}
{{ template "base" }}
```

The critical part is the final `{{ template "base" }}` line. It tells the parser
what root templates need to render here. The `base` block should exist in the
layout templates.

The basic idea of the execution is that we create a clone of the layout Template structure
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

With a simple extension of the Go's standard template system, we can get
a template engine that can be used in web applications similarly to other web frameworks
template systems (e.g. Django).

The `github.com/szferi/codenotes/go-template-engine/` includes an `engine` module that packs
the above ideas into a module.
