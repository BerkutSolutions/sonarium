package http

import (
	"io/fs"
	"net/http"

	guifs "music-server/gui"
)

type Handler struct {
	static http.Handler
	files  fs.FS
}

func NewHandler() *Handler {
	sub, err := fs.Sub(guifs.Assets, "static")
	if err != nil {
		panic(err)
	}
	return &Handler{
		static: http.FileServer(http.FS(sub)),
		files:  sub,
	}
}

func (h *Handler) StaticFiles() http.Handler {
	return h.static
}

func (h *Handler) ServeShell(w http.ResponseWriter, r *http.Request) {
	content, err := fs.ReadFile(h.files, "index.html")
	if err != nil {
		http.Error(w, "ui shell not found", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(content)
}
