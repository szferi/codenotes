//go:build allow_embed

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
