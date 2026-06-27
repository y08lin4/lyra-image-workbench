package retention

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/promptsquare"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

func TestCleanupConservativelyRemovesOnlyExpiredTemporaryAssets(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	outputRoot := filepath.Join(root, "outputs")
	now := time.Date(2026, 6, 27, 12, 0, 0, 0, time.Local)
	oldTime := now.AddDate(0, 0, -40)

	spaceStore, err := spaces.NewFileStore(dataDir)
	if err != nil {
		t.Fatalf("NewFileStore() error = %v", err)
	}
	session, err := spaceStore.CreateOrOpenByPassword("R7!Retention#Vault$2026")
	if err != nil {
		t.Fatalf("CreateOrOpenByPassword() error = %v", err)
	}
	token := session.Token
	jobStore := jobs.NewStore(spaceStore)
	promptSquareStore, err := promptsquare.NewStore(dataDir)
	if err != nil {
		t.Fatalf("promptsquare.NewStore() error = %v", err)
	}

	oldDate := "2026-05-01"
	recentDate := "2026-06-26"
	expiredOutput := writeOldFile(t, filepath.Join(outputRoot, token, oldDate, "expired.png"), "expired", oldTime)
	protectedOutput := writeOldFile(t, filepath.Join(outputRoot, token, oldDate, "protected.png"), "protected", oldTime)
	squareProtectedOutput := writeOldFile(t, filepath.Join(outputRoot, token, oldDate, "square-protected.png"), "square-protected", oldTime)
	recentOutput := writeOldFile(t, filepath.Join(outputRoot, token, recentDate, "recent.png"), "recent", now)

	spaceDir, err := spaceStore.SpaceDir(token)
	if err != nil {
		t.Fatalf("SpaceDir() error = %v", err)
	}
	expiredUploadMeta := writeUploadPair(t, filepath.Join(spaceDir, "uploads"), "ref_expired", "ref_expired.png", oldTime)
	activeUploadMeta := writeUploadPair(t, filepath.Join(spaceDir, "uploads"), "ref_active", "ref_active.png", oldTime)

	oldRefDir := filepath.Join(spaceDir, "job_refs", "img_old")
	activeRefDir := filepath.Join(spaceDir, "job_refs", "img_active")
	if err := os.MkdirAll(oldRefDir, 0o700); err != nil {
		t.Fatalf("mkdir old refs: %v", err)
	}
	if err := os.MkdirAll(activeRefDir, 0o700); err != nil {
		t.Fatalf("mkdir active refs: %v", err)
	}

	mustSaveJob(t, jobStore, jobs.Job{
		ID:         "img_recent_protected",
		SpaceToken: token,
		Status:     jobs.StatusSucceeded,
		Results: []jobs.Result{{
			Index:          0,
			OK:             true,
			Status:         jobs.StatusSucceeded,
			OutputDate:     oldDate,
			OutputFileName: "protected.png",
		}},
		CreatedAt:  now.AddDate(0, 0, -2),
		UpdatedAt:  now.AddDate(0, 0, -2),
		FinishedAt: now.AddDate(0, 0, -2),
	})
	mustSaveJob(t, jobStore, jobs.Job{
		ID:         "img_active",
		SpaceToken: token,
		Status:     jobs.StatusRunning,
		UploadIDs:  []string{"ref_active"},
		CreatedAt:  oldTime,
		UpdatedAt:  oldTime,
	})
	mustSaveJob(t, jobStore, jobs.Job{
		ID:         "img_square",
		SpaceToken: token,
		Status:     jobs.StatusSucceeded,
		Results: []jobs.Result{{
			Index:          0,
			OK:             true,
			Status:         jobs.StatusSucceeded,
			OutputDate:     oldDate,
			OutputFileName: "square-protected.png",
		}},
		CreatedAt:  oldTime,
		UpdatedAt:  oldTime,
		FinishedAt: oldTime,
	})

	mustSaveJob(t, jobStore, jobs.Job{
		ID:         "img_old",
		SpaceToken: token,
		Status:     jobs.StatusSucceeded,
		CreatedAt:  oldTime,
		UpdatedAt:  oldTime,
		FinishedAt: oldTime,
	})

	squareSource := writeOldFile(t, filepath.Join(root, "square-source.png"), "square", oldTime)
	squareItem, err := promptSquareStore.SubmitFromResult(promptsquare.SubmitFromResultRequest{
		Title:           "square",
		Prompt:          "square prompt",
		SourceTaskID:    "img_square",
		SourceImagePath: squareSource,
		SourceImageMime: "image/png",
	})
	if err != nil {
		t.Fatalf("SubmitFromResult() error = %v", err)
	}
	squareImage := filepath.Join(dataDir, "prompt_square", "images", filepath.Base(squareItem.ImageURL))

	report, err := Cleanup(Config{
		OutputRoot:   outputRoot,
		Spaces:       spaceStore,
		Jobs:         jobStore,
		PromptSquare: promptSquareStore,
		Now:          func() time.Time { return now },
	})
	if err != nil {
		t.Fatalf("Cleanup() error = %v", err)
	}
	if report.OutputFilesDeleted != 1 || report.UploadsDeleted != 1 || report.JobRefDirsDeleted != 1 {
		t.Fatalf("unexpected cleanup report: %+v", report)
	}

	assertMissing(t, expiredOutput)
	assertExists(t, protectedOutput)
	assertExists(t, squareProtectedOutput)
	assertExists(t, recentOutput)
	assertMissing(t, expiredUploadMeta)
	assertExists(t, activeUploadMeta)
	assertMissing(t, oldRefDir)
	assertExists(t, activeRefDir)
	assertExists(t, squareImage)
}

func mustSaveJob(t *testing.T, store *jobs.Store, job jobs.Job) {
	t.Helper()
	if err := store.Save(job); err != nil {
		t.Fatalf("Save(%s) error = %v", job.ID, err)
	}
}

func writeOldFile(t *testing.T, path string, body string, ts time.Time) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
	if err := os.Chtimes(path, ts, ts); err != nil {
		t.Fatalf("chtimes %s: %v", path, err)
	}
	return path
}

func writeUploadPair(t *testing.T, uploadDir string, id string, fileName string, ts time.Time) string {
	t.Helper()
	imagePath := writeOldFile(t, filepath.Join(uploadDir, fileName), "upload", ts)
	metaPath := filepath.Join(uploadDir, id+".json")
	body := `{"id":"` + id + `","fileName":"` + fileName + `","createdAt":"` + ts.Format(time.RFC3339) + `"}`
	if err := os.WriteFile(metaPath, []byte(body), 0o600); err != nil {
		t.Fatalf("write upload meta %s: %v", metaPath, err)
	}
	if err := os.Chtimes(metaPath, ts, ts); err != nil {
		t.Fatalf("chtimes upload meta %s: %v", metaPath, err)
	}
	if err := os.Chtimes(filepath.Dir(imagePath), ts, ts); err != nil {
		t.Fatalf("chtimes upload dir: %v", err)
	}
	return metaPath
}

func assertExists(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected %s to exist: %v", path, err)
	}
}

func assertMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("expected %s to be removed, stat err=%v", path, err)
	}
}
