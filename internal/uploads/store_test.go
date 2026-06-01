package uploads

import (
	"bytes"
	"fmt"
	"mime/multipart"
	"net/http/httptest"
	"net/textproto"
	"testing"

	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

func TestSaveListGetDeleteReferenceImages(t *testing.T) {
	store, token := newTestUploadStore(t)
	headers := multipartHeaders(t, []testUploadFile{{
		Name:        "cat.png",
		ContentType: "image/png",
		Data:        pngBytes(),
	}})

	saved, err := store.SaveReferenceImages(token, headers)
	if err != nil {
		t.Fatalf("SaveReferenceImages() error = %v", err)
	}
	if len(saved) != 1 || saved[0].ID == "" || saved[0].Mime != "image/png" {
		t.Fatalf("unexpected saved uploads: %+v", saved)
	}

	listed, err := store.ListReferenceImages(token)
	if err != nil {
		t.Fatalf("ListReferenceImages() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != saved[0].ID {
		t.Fatalf("unexpected list: %+v", listed)
	}

	item, path, err := store.GetReferenceImage(token, saved[0].ID)
	if err != nil {
		t.Fatalf("GetReferenceImage() error = %v", err)
	}
	if item.FileName == "" || path == "" {
		t.Fatalf("missing file metadata: item=%+v path=%q", item, path)
	}

	if err := store.DeleteReferenceImage(token, saved[0].ID); err != nil {
		t.Fatalf("DeleteReferenceImage() error = %v", err)
	}
	listed, err = store.ListReferenceImages(token)
	if err != nil {
		t.Fatalf("ListReferenceImages(after delete) error = %v", err)
	}
	if len(listed) != 0 {
		t.Fatalf("expected empty list after delete, got %+v", listed)
	}
}

func TestReferenceUploadLimits(t *testing.T) {
	store, token := newTestUploadStore(t)

	tooMany := make([]testUploadFile, MaxReferenceImages+1)
	for i := range tooMany {
		tooMany[i] = testUploadFile{Name: fmt.Sprintf("img-%d.png", i), ContentType: "image/png", Data: pngBytes()}
	}
	if _, err := store.SaveReferenceImages(token, multipartHeaders(t, tooMany)); !IsUploadCode(err, "REFERENCE_IMAGE_TOO_MANY") {
		t.Fatalf("expected too many error, got %v", err)
	}

	oversized := bytes.Repeat([]byte{0xff}, MaxReferenceImageBytes+1)
	headers := multipartHeaders(t, []testUploadFile{{Name: "large.jpg", ContentType: "image/jpeg", Data: oversized}})
	if _, err := store.SaveReferenceImages(token, headers); !IsUploadCode(err, "REFERENCE_IMAGE_TOO_LARGE") {
		t.Fatalf("expected too large error, got %v", err)
	}

	spoofed := multipartHeaders(t, []testUploadFile{{Name: "fake.png", ContentType: "image/png", Data: []byte("not really an image")}})
	if _, err := store.SaveReferenceImages(token, spoofed); !IsUploadCode(err, "REFERENCE_IMAGE_TYPE_UNSUPPORTED") {
		t.Fatalf("expected unsupported type error for spoofed image, got %v", err)
	}
}

func TestReferenceUploadTotalLimitIncludesExistingImages(t *testing.T) {
	store, token := newTestUploadStore(t)
	for i := 0; i < MaxReferenceImages; i++ {
		headers := multipartHeaders(t, []testUploadFile{{Name: fmt.Sprintf("img-%d.png", i), ContentType: "image/png", Data: pngBytes()}})
		if _, err := store.SaveReferenceImages(token, headers); err != nil {
			t.Fatalf("SaveReferenceImages(%d) error = %v", i, err)
		}
	}
	extra := multipartHeaders(t, []testUploadFile{{Name: "extra.png", ContentType: "image/png", Data: pngBytes()}})
	if _, err := store.SaveReferenceImages(token, extra); !IsUploadCode(err, "REFERENCE_IMAGE_TOO_MANY") {
		t.Fatalf("expected total limit error, got %v", err)
	}
}

func newTestUploadStore(t *testing.T) (*Store, string) {
	t.Helper()
	spaceStore, err := spaces.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Blue#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	return NewStore(spaceStore), session.Token
}

type testUploadFile struct {
	Name        string
	ContentType string
	Data        []byte
}

func multipartHeaders(t *testing.T, files []testUploadFile) []*multipart.FileHeader {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for _, file := range files {
		header := make(textproto.MIMEHeader)
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="image[]"; filename="%s"`, file.Name))
		header.Set("Content-Type", file.ContentType)
		part, err := writer.CreatePart(header)
		if err != nil {
			t.Fatalf("CreatePart() error = %v", err)
		}
		if _, err := part.Write(file.Data); err != nil {
			t.Fatalf("part.Write() error = %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("writer.Close() error = %v", err)
	}
	req := httptest.NewRequest("POST", "/upload", &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	if err := req.ParseMultipartForm(MaxReferenceUploadBytes); err != nil {
		t.Fatalf("ParseMultipartForm() error = %v", err)
	}
	return req.MultipartForm.File["image[]"]
}

func pngBytes() []byte {
	return append([]byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}, bytes.Repeat([]byte{0}, 64)...)
}

func IsUploadCode(err error, code string) bool {
	var uploadErr UploadError
	return AsUploadError(err, &uploadErr) && uploadErr.Code == code
}
