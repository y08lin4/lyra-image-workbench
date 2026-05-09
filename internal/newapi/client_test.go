package newapi

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestGenerateParsesB64JSON(t *testing.T) {
	want := []byte("png-bytes")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/images/generations" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk-test" {
			t.Fatalf("Authorization = %q", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		if body["model"] != "gpt-image-2" || body["prompt"] != "cat" || body["n"].(float64) != 1 || body["quality"] != "high" {
			t.Fatalf("unexpected request body: %+v", body)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{
			"b64_json":       base64.StdEncoding.EncodeToString(want),
			"revised_prompt": "a revised cat",
			"size":           "1024x1024",
			"quality":        "high",
			"output_format":  "png",
		}}})
	}))
	defer server.Close()

	client := NewClient()
	client.httpClient = server.Client()
	image, err := client.Generate(context.Background(), Request{
		Mode:       "text-to-image",
		BaseURL:    server.URL + "/v1",
		APIKey:     "sk-test",
		Model:      "gpt-image-2",
		Prompt:     "cat",
		Size:       "1024x1024",
		Quality:    "high",
		TimeoutSec: 60,
	})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if string(image.Bytes) != string(want) || image.Mime != "image/png" {
		t.Fatalf("Generate() image = %+v", image)
	}
	if image.RevisedPrompt != "a revised cat" || image.ActualSize != "1024x1024" || image.ActualQuality != "high" || image.OutputFormat != "png" {
		t.Fatalf("Generate() metadata = %+v", image)
	}
}

func TestGenerateParsesRawImageResponse(t *testing.T) {
	want := []byte("raw-image")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/png")
		_, _ = w.Write(want)
	}))
	defer server.Close()

	client := NewClient()
	client.httpClient = server.Client()
	image, err := client.Generate(context.Background(), Request{Mode: "text-to-image", BaseURL: server.URL, APIKey: "sk", Model: "gpt-image-2", Prompt: "raw", TimeoutSec: 60})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if string(image.Bytes) != string(want) || image.Mime != "image/png" {
		t.Fatalf("Generate() image = %+v", image)
	}
}

func TestGenerateDownloadsURLResponse(t *testing.T) {
	want := []byte("downloaded-image")
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/v1/images/generations", func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"url": server.URL + "/image.webp"}}})
	})
	mux.HandleFunc("/image.webp", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "image/webp")
		_, _ = w.Write(want)
	})

	client := NewClient()
	client.httpClient = server.Client()
	image, err := client.Generate(context.Background(), Request{Mode: "text-to-image", BaseURL: server.URL + "/v1", APIKey: "sk", Model: "gpt-image-2", Prompt: "url", TimeoutSec: 60})
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if string(image.Bytes) != string(want) || image.Mime != "image/webp" {
		t.Fatalf("Generate() image = %+v", image)
	}
}

func TestEditImageSendsMultipartImages(t *testing.T) {
	dir := t.TempDir()
	imagePath := filepath.Join(dir, "input.png")
	if err := os.WriteFile(imagePath, []byte("input-image"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/images/edits" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			t.Fatalf("ParseMultipartForm() error = %v", err)
		}
		if r.FormValue("model") != "gpt-image-2" || r.FormValue("prompt") != "edit prompt" || r.FormValue("response_format") != "b64_json" || r.FormValue("quality") != "medium" {
			t.Fatalf("unexpected form values: %+v", r.MultipartForm.Value)
		}
		files := r.MultipartForm.File["image[]"]
		if len(files) != 1 {
			t.Fatalf("expected one image[] file, got %d", len(files))
		}
		if files[0].Header.Get("Content-Type") != "image/png" {
			t.Fatalf("image part content type = %q", files[0].Header.Get("Content-Type"))
		}
		file, err := files[0].Open()
		if err != nil {
			t.Fatalf("open uploaded file: %v", err)
		}
		defer file.Close()
		data, _ := io.ReadAll(file)
		if string(data) != "input-image" {
			t.Fatalf("uploaded image bytes = %q", data)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString([]byte("edited"))}}})
	}))
	defer server.Close()

	client := NewClient()
	client.httpClient = server.Client()
	image, err := client.Generate(context.Background(), Request{
		Mode:       "image-to-image",
		BaseURL:    server.URL,
		APIKey:     "sk",
		Model:      "gpt-image-2",
		Prompt:     "edit prompt",
		Quality:    "medium",
		TimeoutSec: 60,
		InputImages: []InputImage{{
			Name: "input.png",
			Path: imagePath,
			Mime: "image/png",
		}},
	})
	if err != nil {
		t.Fatalf("Generate(edit) error = %v", err)
	}
	if string(image.Bytes) != "edited" {
		t.Fatalf("edited bytes = %q", image.Bytes)
	}
}

func TestGenerateHonorsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(2 * time.Second):
			_, _ = w.Write([]byte(`{"data":[]}`))
		}
	}))
	defer server.Close()

	client := NewClient()
	client.httpClient = server.Client()
	_, err := client.Generate(context.Background(), Request{Mode: "text-to-image", BaseURL: server.URL, APIKey: "sk", Model: "gpt-image-2", Prompt: "slow", TimeoutSec: 1})
	if err == nil || !strings.Contains(err.Error(), "context deadline exceeded") {
		t.Fatalf("expected timeout error, got %v", err)
	}
}
