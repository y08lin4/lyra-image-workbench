package api

import (
	"net/http"

	"github.com/y08lin4/image-Workbench-Localhost-Version/internal/spaces"
)

func writeSpaceError(w http.ResponseWriter, err error) {
	status := http.StatusUnauthorized
	code := "SPACE_AUTH_ERROR"
	message := err.Error()
	var validationErr spaces.ValidationError
	if spaces.AsValidationError(err, &validationErr) {
		code = validationErr.Code
		message = validationErr.Chinese
		if code == "SPACE_NOT_FOUND" {
			status = http.StatusNotFound
		} else {
			status = http.StatusBadRequest
		}
	}
	writeError(w, status, code, message)
}
