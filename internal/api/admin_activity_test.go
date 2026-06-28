package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/y08lin4/lyra-image-workbench/internal/activitylog"
)

func TestAdminActivityListsAndFiltersLogs(t *testing.T) {
	env := newTestAPIEnv(t)
	userToken := createNamedUserSession(t, env.Router, "activityuser01", "R7!Activity#Vault$2026", "")
	adminToken := createAdminToken(t, env.Router)

	doAdminJSON(t, env.Router, http.MethodPost, "/api/admin/users/credits/add", adminToken, map[string]any{
		"username": "activityuser01",
		"amount":   5,
		"reason":   "activity test grant",
	})

	body := doAdminJSON(t, env.Router, http.MethodGet, "/api/admin/activity?type=admin_credit_grant&level=info&limit=10", adminToken, nil)
	var grantResponse struct {
		Activity []activitylog.Entry `json:"activity"`
	}
	if err := json.Unmarshal([]byte(body), &grantResponse); err != nil {
		t.Fatalf("decode admin activity response: %v body=%s", err, body)
	}
	if len(grantResponse.Activity) != 1 || grantResponse.Activity[0].Type != activitylog.TypeAdminCreditGrant || grantResponse.Activity[0].Username != "activityuser01" {
		t.Fatalf("filtered admin activity mismatch: %+v body=%s", grantResponse.Activity, body)
	}

	body = doAdminJSON(t, env.Router, http.MethodGet, "/api/admin/activity?type=user_registration&limit=10", adminToken, nil)
	if !strings.Contains(body, "activityuser01") || !strings.Contains(body, "user_registration") {
		t.Fatalf("registration activity missing: %s", body)
	}

	code, unauthorizedBody := doJSONStatus(t, env.Router, http.MethodGet, "/api/admin/activity", userToken, nil, "")
	if code != http.StatusUnauthorized || !strings.Contains(unauthorizedBody, "ADMIN_AUTH_REQUIRED") {
		t.Fatalf("non-admin activity access code=%d body=%s", code, unauthorizedBody)
	}
}
