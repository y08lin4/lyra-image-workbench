package api

import (
	"net/http"
	"strings"

	"github.com/y08lin4/lyra-image-workbench/internal/users"
)

const userStorageTokenHeader = "X-Space-Token"

func withUserAuth(store *users.Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !requiresUserAuth(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		session, ok := currentUserSession(store, r)
		if !ok {
			writeError(w, http.StatusUnauthorized, "USER_AUTH_REQUIRED", "请先登录")
			return
		}
		r.Header.Set(userStorageTokenHeader, session.StorageToken)
		r.Header.Set("X-User-Name", session.User.Username)
		next.ServeHTTP(w, r)
	})
}

func requiresUserAuth(path string) bool {
	switch {
	case path == "/api/config":
		return true
	case path == "/api/stats":
		return true
	case strings.HasPrefix(path, "/api/uploads/"):
		return true
	case strings.HasPrefix(path, "/api/background-tasks"):
		return true
	case path == "/api/prompt-library" || strings.HasPrefix(path, "/api/prompt-library/"):
		return true
	case strings.HasPrefix(path, "/api/prompt-tools/"):
		return true
	case strings.HasPrefix(path, "/api/users/2fa/"):
		return true
	case strings.HasPrefix(path, "/outputs/"):
		return true
	default:
		return false
	}
}
