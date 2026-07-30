package web

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

//go:embed all:dist
var embedded embed.FS

var assets = mustSub(embedded, "dist")

func Register(e *core.ServeEvent) {
	handler := func(request *core.RequestEvent) error {
		return serve(request, request.Request.URL.Path)
	}
	e.Router.GET("/{path...}", handler)
}

func serve(e *core.RequestEvent, requestPath string) error {
	setSecurityHeaders(e)
	clean := strings.TrimPrefix(path.Clean("/"+requestPath), "/")
	if clean == "." || clean == "" {
		clean = "index.html"
	}
	if strings.HasPrefix(clean, "api/") || clean == "api" || strings.HasPrefix(clean, "_/") || clean == "_" {
		return e.NotFoundError("Not found", nil)
	}
	data, err := fs.ReadFile(assets, clean)
	if err != nil {
		if path.Ext(clean) != "" {
			return e.NotFoundError("Asset not found", nil)
		}
		clean = "index.html"
		data, err = fs.ReadFile(assets, clean)
		if err != nil {
			return e.NotFoundError("Frontend is not available", err)
		}
	}
	if strings.HasPrefix(clean, "assets/") {
		e.Response.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	} else {
		e.Response.Header().Set("Cache-Control", "no-cache")
	}
	contentType := mime.TypeByExtension(path.Ext(clean))
	if contentType == "" {
		contentType = http.DetectContentType(data)
	}
	if e.Request.Method == http.MethodHead {
		e.Response.Header().Set("Content-Type", contentType)
		return e.NoContent(http.StatusOK)
	}
	return e.Blob(http.StatusOK, contentType, data)
}

func setSecurityHeaders(e *core.RequestEvent) {
	headers := e.Response.Header()
	headers.Set("X-Content-Type-Options", "nosniff")
	headers.Set("X-Frame-Options", "SAMEORIGIN")
	headers.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	headers.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	headers.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; font-src 'self'; img-src 'self' data: blob:; connect-src 'self'; object-src 'none'; base-uri 'self'; frame-ancestors 'self'")
}
func mustSub(root fs.FS, dir string) fs.FS {
	sub, err := fs.Sub(root, dir)
	if err != nil {
		panic(err)
	}
	return sub
}
