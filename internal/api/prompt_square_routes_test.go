package api

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
)

func TestPromptSquareRoutesRequireLogin(t *testing.T) {
	router := newTestRouter(t)

	for _, tc := range []struct {
		method string
		path   string
		body   string
	}{
		{http.MethodPost, "/api/prompt-square/from-result", `{}`},
		{http.MethodPost, "/api/prompt-square/items/prompt_missing/like", `{}`},
		{http.MethodGet, "/api/prompt-square/daily", ""},
		{http.MethodGet, "/api/prompt-square/mine", ""},
	} {
		req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
		if tc.body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		res := httptest.NewRecorder()
		router.ServeHTTP(res, req)
		if res.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s without login code=%d body=%s", tc.method, tc.path, res.Code, res.Body.String())
		}
	}
}

func TestPromptSquareRoutesUseAuthenticatedUserAndResultDependencies(t *testing.T) {
	env := newTestAPIEnv(t)
	token := createTestSession(t, env.Router)
	session, ok := env.Users.Current(token)
	if !ok {
		t.Fatal("test session missing")
	}

	legacyItemID := createPromptSquareUploadItem(t, env.Router, token)

	saved, err := env.Output.Save(session.StorageToken, "img_prompt_square_route", 0, []byte("prompt-square-route-image"), "image/png")
	if err != nil {
		t.Fatalf("Output.Save() error = %v", err)
	}
	result := jobs.NewResult(0, jobs.StatusSucceeded, "")
	result.ImageURL = saved.URL
	result.OutputDate = saved.Date
	result.OutputFileName = saved.FileName
	result.Mime = saved.Mime
	result.ActualQuality = "high"
	result.OutputFormat = "png"
	now := time.Now()
	job := jobs.Job{
		ID:           "img_prompt_square_route",
		SpaceToken:   session.StorageToken,
		Provider:     "image-2",
		Model:        "gpt-image-2",
		Mode:         jobs.ModeTextToImage,
		Prompt:       "route prompt",
		Ratio:        "1:1",
		Resolution:   "standard",
		Quality:      "auto",
		OutputFormat: "png",
		Count:        1,
		Concurrency:  1,
		Progress:     100,
		Results:      []jobs.Result{result},
		CreatedAt:    now,
		UpdatedAt:    now,
		FinishedAt:   now,
	}
	jobs.ApplyStatus(&job, jobs.StatusSucceeded)
	jobs.ApplyStage(&job, jobs.StageSucceeded)
	if err := env.Jobs.Save(job); err != nil {
		t.Fatalf("Jobs.Save() error = %v", err)
	}

	fromResultBody := doJSON(t, env.Router, http.MethodPost, "/api/prompt-square/from-result", token, map[string]any{
		"taskId":     job.ID,
		"imageIndex": 0,
		"title":      "from result route",
		"tags":       []string{"router"},
	})
	fromResultID := decodePromptSquareItemID(t, fromResultBody)
	if fromResultID == "" {
		t.Fatalf("from-result response missing item id: %s", fromResultBody)
	}

	likeBody := doJSON(t, env.Router, http.MethodPost, "/api/prompt-square/items/"+fromResultID+"/like", token, map[string]bool{"liked": true})
	if !strings.Contains(likeBody, `"likedByMe":true`) || !strings.Contains(likeBody, `"likeCount":1`) {
		t.Fatalf("like response missing authenticated user state: %s", likeBody)
	}

	mineBody := doJSON(t, env.Router, http.MethodGet, "/api/prompt-square/mine", token, nil)
	if !strings.Contains(mineBody, fromResultID) || !strings.Contains(mineBody, `"author":"testuser01"`) {
		t.Fatalf("mine response missing authenticated user's item: %s", mineBody)
	}

	dailyBody := doJSON(t, env.Router, http.MethodGet, "/api/prompt-square/daily", token, nil)
	if !strings.Contains(dailyBody, fromResultID) || !strings.Contains(dailyBody, `"dailyRank":`) {
		t.Fatalf("daily response missing from-result item: %s", dailyBody)
	}

	listBody := doJSON(t, env.Router, http.MethodGet, "/api/prompt-square/items", token, nil)
	if !strings.Contains(listBody, legacyItemID) || !strings.Contains(listBody, fromResultID) {
		t.Fatalf("legacy list response missing prompt-square items: %s", listBody)
	}
}

func createPromptSquareUploadItem(t *testing.T, router http.Handler, token string) string {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range map[string]string{
		"title":        "legacy upload route",
		"prompt":       "legacy upload prompt",
		"model":        "gpt-image-2",
		"ratio":        "1:1",
		"quality":      "auto",
		"outputFormat": "png",
	} {
		if err := writer.WriteField(key, value); err != nil {
			t.Fatalf("WriteField(%s) error = %v", key, err)
		}
	}
	part, err := writer.CreateFormFile("image", "fixture.gif")
	if err != nil {
		t.Fatalf("CreateFormFile() error = %v", err)
	}
	if _, err := part.Write([]byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\x00\x00\x00\xff\xff\xff!\xf9\x04\x00\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")); err != nil {
		t.Fatalf("write gif fixture error = %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("multipart close error = %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/prompt-square/items", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.AddCookie(&http.Cookie{Name: userSessionCookie, Value: token})
	res := httptest.NewRecorder()
	router.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Fatalf("POST /api/prompt-square/items code=%d body=%s", res.Code, res.Body.String())
	}
	return decodePromptSquareItemID(t, res.Body.String())
}

func decodePromptSquareItemID(t *testing.T, body string) string {
	t.Helper()
	var payload struct {
		Item struct {
			ID string `json:"id"`
		} `json:"item"`
	}
	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		t.Fatalf("decode prompt-square item response: %v body=%s", err, body)
	}
	return payload.Item.ID
}
