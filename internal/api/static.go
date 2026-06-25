package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type StaticHandler struct {
	dir string
}

func NewStaticHandler(dir string) StaticHandler { return StaticHandler{dir: dir} }

func (h StaticHandler) Serve(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		http.NotFound(w, r)
		return
	}
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "api" || strings.HasPrefix(path, "api/") || path == "v1" || strings.HasPrefix(path, "v1/") || path == "outputs" || strings.HasPrefix(path, "outputs/") {
		http.NotFound(w, r)
		return
	}
	if path == "" {
		path = "index.html"
	}
	file := filepath.Join(h.dir, filepath.Clean(path))
	root, _ := filepath.Abs(h.dir)
	abs, _ := filepath.Abs(file)
	if abs != root && !strings.HasPrefix(abs, root+string(filepath.Separator)) {
		http.NotFound(w, r)
		return
	}
	if info, err := os.Stat(abs); err == nil && !info.IsDir() {
		http.ServeFile(w, r, abs)
		return
	}
	index := filepath.Join(h.dir, "index.html")
	if _, err := os.Stat(index); err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("前端文件不存在，请先运行：cd web && npm run build"))
		return
	}
	http.ServeFile(w, r, index)
}
