package api

import (
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/jobs"
)

const (
	runtimeAPIKeyHeader       = "X-Image-Workbench-API-Key"
	runtimeBananaAPIKeyHeader = "X-Image-Workbench-Banana-API-Key"
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
