package api

import (
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/config"
)

type Dependencies struct {
	Config config.Config
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	health := NewHealthHandler(deps.Config)

	mux.HandleFunc("GET /api/health", health.ServeHTTP)

	return withCommonHeaders(mux)
}

func withCommonHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		next.ServeHTTP(w, r)
	})
}
