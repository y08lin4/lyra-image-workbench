package api

import (
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
)

const (
	runtimeAPIKeyHeader        = "X-Image-Workbench-API-Key"
	runtimeBananaAPIKeyHeader  = "X-Image-Workbench-Banana-API-Key"
	runtimeMiniMaxAPIKeyHeader = "X-Image-Workbench-Minimax-API-Key"
)

func runtimeSecretsFromRequest(r *http.Request) jobs.RuntimeSecrets {
	return jobs.RuntimeSecrets{
		APIKey:       r.Header.Get(runtimeAPIKeyHeader),
		BananaAPIKey: r.Header.Get(runtimeBananaAPIKeyHeader),
	}
}

func runtimeAPIKeyFromRequest(r *http.Request) string {
	return r.Header.Get(runtimeAPIKeyHeader)
}

func minimaxAPIKeyFromRequest(r *http.Request) string {
	return r.Header.Get(runtimeMiniMaxAPIKeyHeader)
}
