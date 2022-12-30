+++
summary: How to use `go:embed` directive conditionaly
is_pinned: True
is_published: True
+++

# Conditional file embeding in Go

One technique to simplify go web application deployment is to embed files and folders into a go binary at compile-time.
Since 1.16 version go provides the embed module in the standard library that help to achive this goal without any
additional tools using the `//go:embed` compiler directive. However, during development you want your assets and
templates read and rendered for every browser refresh or changes without not necessary recompile the whole application.
This helps when you focused to update your CSS or HTML part of the application and do not change the application logic
code.

Unfortunatelly there is no native way to define when you want to embed or not given resources at compile-time, but can
simulate the effect using compiler build tags.

The general idea is that we define a build tag called `allow_embed` and we create two file, one that defines a default
behaviour and only included build process if the tag is not present (lets call this file `embedding_not_allowed.go`) and the other one that only included to the build process if this tag is present and it contains the `//go:embed` directive (lets call this file `embedding_allowed.go`). Both file defines the same variables with different values.

In the followings I show how to use the technique with the two most common resourcs web application have: assets,
like css, javascript and image files and HTML templates. Therefore I assume that there is an `assets` and a `templates`
directory in the root of the source code. It is important that the go code that contains the `//go:embed` directive
must be in the same directory where these resource directories are.

## Setup build tag

Lets first setup the build tag. The `embedding_allowed.go` file uses directive `// +build allow_embed` to tell the compiler that this file is only should included to the build process if the `allow_embed` build tag is present, while the `embedding_not_allowed.go` file contains the negated version of the same instruction. Both file defines an `IsEmbeddingAllowed()` function which returns true or false respectively and it can be use to define conditional beheviour
in other part of the code.

```go, filename=embedding_allowed.go
// +build allow_embed

package main

func init() {
	IsEmbeddingAllowed = true
}
```

```go, filename=embeddin_not_allowed.go
// +build !allow_embed

package main

func init() {
	IsEmbeddingAllowed = false
}
```

Creating a following simple `main.go` allow as to test our setup

```go, filename=main.go
package main

import "fmt"

func main() {
	fmt.Println(IsEmbeddingAllowed())
}
```

Compiling without any build tag we exept "false" as an output:

```shell, lineno=False
> go build -o m . && ./m
> false
```

While if we define `allow_embed` build tag we expect "true" as an output:

```shell, lineno=False
> go build -o m -tags allow_embed . && ./m
> true
```

## Add Embeded Filesystems

As a next step we add two new variables with the type `embed.FS` to each embedding version.
In the `embedding_allowed.go` files we agument the variables with the appropriate `//go:embed` directive while in the
file `embedding_not_allowed.go` we keep the variables uninitialized since we will not use them later.

We also add code that walk through in each FS and prints it content into the `main.go` to demonstrate the differences.

```go, filename=embedding_allowed.go
// +build allow_embed

package main

import "embed"

//go:embed assets
var assetsFS embed.FS

//go:embed templates
var templatesFS embed.FS

func IsEmbeddingAllowed() bool {
	return true
}
```

```go, filename=embeding_not_allowed.go
// +build !allow_embed

package main

import "embed"

func IsEmbeddingAllowed() bool {
	return false
}
```

```go, filename=main.go

package main

import (
	"fmt"
	"io/fs"
)

func showPath(path string, d fs.DirEntry, err error) error {
    fmt.Println(path)
    return nil
}

func main() {
	fmt.Println(IsEmbeddingAllowed())
	fmt.Println("Assets:")
	fs.WalkDir(assetsFS, ".", showPath)
	fmt.Println("Templates:")
	fs.WalkDir(templatesFS, ".", showPath)
}
```

If we compile the code without tag we should only see empty filesystems while if we set the `allow_embed` tag
the listing should show the current content of the assets and templates repectively. Also you can see the difference
of the size of the compiled executable. You can also test if indeed the content of the assets and templates
directories are embeded into the executable by moving it to somewhere else int the filesystem and after running
it you should still see the content of these directories listed while there are nearwhere close.

## Integrate to web application

While the previous section showed that if you define `allow_embed` compiler tag you get the embed.FS filled up
with the content of the assets and templates directory it fail short if the tag is not defined. Too connect
the embed.FS and fix the default behaviour we define two functions: `GetAssetsHandler()` and `GetTemplatesFS()` and implement them very differently in the `embedding_allowed.go` and the `embedding_not_allowed.go` files.

In the case of `allow_embed` tag not defined these functions implement the default behaviour ignoring the `assetsFS`
and `templatesFS` variables altogether. The go linter will complain about this marking the variables defined but
not used. If indeed we will not use them directly we can remove them from the `embedding_not_allowed.go` file.

In the allowed case the `templatesFS` variable can be used without modification since it implements the `fs.FS` interface.
While the `assetsFS` variable should casted to `http.FS` and wraped with `http.FileServer` function.

Finally the `main.go` code shows a simple way how to use these two new functions to define the behaviour of the "/" and "/assets/" endpoints.

```go, filename=embedding_allowed.go
// +build allow_embed

package main

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed assets
var assetsFS embed.FS

//go:embed templates
var templatesFS embed.FS

func IsEmbeddingAllowed() bool {
	return true
}

func GetAssetHandler() http.Handler {
	return http.FileServer(http.FS(assetsFS))
}

func GetTemplatesFS() fs.FS {
	return templatesFS
}
```

```go, filename=embedding_now_allowed.go
package main

import (
	"io/fs"
	"net/http"
	"os"
)

func IsEmbeddingAllowed() bool {
	return false
}

func GetAssetHandler() http.Handler {
	return http.StripPrefix("/assets/", http.FileServer(http.Dir("./assets")))
}

func GetTemplatesFS() fs.FS {
	return os.DirFS("./templates")
}
```

```go, filename=main.go
package main

import (
	"html/template"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		tpl, err := template.ParseFS(GetTemplatesFS(), "*")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		err = tpl.ExecuteTemplate(w, "home.html", nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	})
	http.Handle("/assets/", GetAssetHandler())
	log.Fatal(http.ListenAndServe("localhost:3000", nil))
}
```

Using the `IsEmbeddingAllowed()` function you can further optimize the system
bacause you can parse the templates outside of the handle function.
