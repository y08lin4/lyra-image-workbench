package api

import (
	"net/http"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
)

const (
	runtimeAPIKeyHeader = "X-Image-Workbench-API-Key"
)

func runtimeSecretsFromRequest(r *http.Request) jobs.RuntimeSecrets {
	return jobs.RuntimeSecrets{
		APIKey: r.Header.Get(runtimeAPIKeyHeader),
	}
}

func runtimeAPIKeyFromRequest(r *http.Request) string {
	return r.Header.Get(runtimeAPIKeyHeader)
}
