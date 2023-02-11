//go:build !allow_embed

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
