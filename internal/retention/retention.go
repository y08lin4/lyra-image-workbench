package retention

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/y08lin4/lyra-image-workbench/internal/jobs"
	"github.com/y08lin4/lyra-image-workbench/internal/promptsquare"
	"github.com/y08lin4/lyra-image-workbench/internal/spaces"
)

const DefaultTTL = 30 * 24 * time.Hour

var outputDateDirRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

type Config struct {
	OutputRoot   string
	Spaces       *spaces.FileStore
	Jobs         *jobs.Store
	PromptSquare *promptsquare.Store
	TTL          time.Duration
	Now          func() time.Time
	Logf         func(format string, args ...any)
}

type Report struct {
	OutputFilesDeleted int
	OutputDirsDeleted  int
	UploadsDeleted     int
	JobRefDirsDeleted  int
	Skipped            int
}

func Cleanup(cfg Config) (Report, error) {
	if cfg.TTL <= 0 {
		cfg.TTL = DefaultTTL
	}
	now := time.Now()
	if cfg.Now != nil {
		now = cfg.Now()
	}
	cutoff := now.Add(-cfg.TTL)

	index, err := buildJobIndex(cfg, cutoff)
	if err != nil {
		return Report{}, err
	}

	var report Report
	if cfg.OutputRoot != "" {
		next, err := cleanupOutputs(cfg.OutputRoot, cutoff, index)
		report.add(next)
		if err != nil {
			return report, err
		}
	}
	if cfg.Spaces != nil {
		next, err := cleanupSpaceTempAssets(cfg.Spaces, cutoff, index)
		report.add(next)
		if err != nil {
			return report, err
		}
	}
	return report, nil
}

func StartDaily(cfg Config) {
	logf := cfg.Logf
	if logf == nil {
		logf = log.Printf
	}
	go func() {
		time.Sleep(30 * time.Second)
		runCleanup(cfg, logf)
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			runCleanup(cfg, logf)
		}
	}()
}

func runCleanup(cfg Config, logf func(format string, args ...any)) {
	report, err := Cleanup(cfg)
	if err != nil {
		logf("30 天清理失败：%v", err)
		return
	}
	if report.OutputFilesDeleted == 0 && report.UploadsDeleted == 0 && report.JobRefDirsDeleted == 0 {
		return
	}
	logf("30 天清理完成：输出图片 %d，上传参考图 %d，任务参考快照 %d，空输出目录 %d，跳过 %d",
		report.OutputFilesDeleted,
		report.UploadsDeleted,
		report.JobRefDirsDeleted,
		report.OutputDirsDeleted,
		report.Skipped,
	)
}

type jobIndex struct {
	jobsBySpace      map[string][]jobs.Job
	protectedOutputs map[string]struct{}
	activeUploads    map[string]map[string]struct{}
	squareTaskIDs    map[string]struct{}
}

func buildJobIndex(cfg Config, cutoff time.Time) (jobIndex, error) {
	index := jobIndex{
		jobsBySpace:      map[string][]jobs.Job{},
		protectedOutputs: map[string]struct{}{},
		activeUploads:    map[string]map[string]struct{}{},
		squareTaskIDs:    map[string]struct{}{},
	}
	if cfg.PromptSquare != nil {
		ids, err := cfg.PromptSquare.SourceTaskIDs()
		if err != nil {
			return index, err
		}
		index.squareTaskIDs = ids
	}

	if cfg.Jobs == nil {
		return index, nil
	}
	bySpace, err := cfg.Jobs.AllSpacesJobs()
	if err != nil {
		return index, err
	}
	index.jobsBySpace = bySpace
	for token, list := range bySpace {
		for _, job := range list {
			_, submittedToSquare := index.squareTaskIDs[job.ID]
			if submittedToSquare || !eligibleOldFinalJob(job, cutoff) {
				for _, result := range job.Results {
					if result.OutputDate != "" && result.OutputFileName != "" {
						index.protectedOutputs[outputKey(token, result.OutputDate, result.OutputFileName)] = struct{}{}
					}
				}
				for _, uploadID := range job.UploadIDs {
					addActiveUpload(index.activeUploads, token, uploadID)
				}
			}
		}
	}
	return index, nil
}

func cleanupOutputs(root string, cutoff time.Time, index jobIndex) (Report, error) {
	var report Report
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, err
	}
	for _, tokenEntry := range entries {
		if !tokenEntry.IsDir() {
			continue
		}
		token, err := spaces.NormalizeToken(tokenEntry.Name())
		if err != nil {
			report.Skipped++
			continue
		}
		tokenDir := filepath.Join(root, token)
		dateEntries, err := os.ReadDir(tokenDir)
		if err != nil {
			return report, err
		}
		for _, dateEntry := range dateEntries {
			if !dateEntry.IsDir() || !outputDateDirRe.MatchString(dateEntry.Name()) {
				report.Skipped++
				continue
			}
			day, err := time.ParseInLocation("2006-01-02", dateEntry.Name(), time.Local)
			if err != nil || !day.Before(dayStart(cutoff)) {
				continue
			}
			dateDir := filepath.Join(tokenDir, dateEntry.Name())
			if err := cleanupOutputDateDir(dateDir, token, dateEntry.Name(), cutoff, index, &report); err != nil {
				return report, err
			}
			if removeEmptyDir(dateDir) {
				report.OutputDirsDeleted++
			}
		}
		_ = removeEmptyDir(tokenDir)
	}
	return report, nil
}

func cleanupOutputDateDir(dir string, token string, date string, cutoff time.Time, index jobIndex, report *Report) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			report.Skipped++
			continue
		}
		name := entry.Name()
		if _, ok := index.protectedOutputs[outputKey(token, date, name)]; ok {
			report.Skipped++
			continue
		}
		path := filepath.Join(dir, name)
		if !isOldFile(path, cutoff) {
			report.Skipped++
			continue
		}
		if err := os.Remove(path); err != nil {
			return err
		}
		report.OutputFilesDeleted++
	}
	return nil
}

func cleanupSpaceTempAssets(store *spaces.FileStore, cutoff time.Time, index jobIndex) (Report, error) {
	var report Report
	tokens, err := store.ListTokens()
	if err != nil {
		return report, err
	}
	for _, token := range tokens {
		spaceDir, err := store.SpaceDir(token)
		if err != nil {
			report.Skipped++
			continue
		}
		next, err := cleanupUploads(filepath.Join(spaceDir, "uploads"), token, cutoff, index.activeUploads)
		report.add(next)
		if err != nil {
			return report, err
		}
		next, err = cleanupJobRefs(filepath.Join(spaceDir, "job_refs"), token, cutoff, index.jobsBySpace[token])
		report.add(next)
		if err != nil {
			return report, err
		}
	}
	return report, nil
}

func cleanupUploads(uploadDir string, token string, cutoff time.Time, active map[string]map[string]struct{}) (Report, error) {
	var report Report
	entries, err := os.ReadDir(uploadDir)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		metaPath := filepath.Join(uploadDir, entry.Name())
		meta, err := readReferenceMeta(metaPath)
		if err != nil {
			report.Skipped++
			continue
		}
		if _, ok := active[token][meta.ID]; ok {
			report.Skipped++
			continue
		}
		createdAt, err := time.Parse(time.RFC3339, strings.TrimSpace(meta.CreatedAt))
		if err != nil || !createdAt.Before(cutoff) {
			continue
		}
		imagePath := filepath.Join(uploadDir, filepath.Base(meta.FileName))
		if !isOldFile(metaPath, cutoff) || (fileExists(imagePath) && !isOldFile(imagePath, cutoff)) {
			report.Skipped++
			continue
		}
		_ = os.Remove(imagePath)
		if err := os.Remove(metaPath); err != nil && !os.IsNotExist(err) {
			return report, err
		}
		report.UploadsDeleted++
	}
	return report, nil
}

func cleanupJobRefs(root string, token string, cutoff time.Time, list []jobs.Job) (Report, error) {
	var report Report
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return report, nil
		}
		return report, err
	}
	byID := map[string]jobs.Job{}
	for _, job := range list {
		byID[job.ID] = job
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			report.Skipped++
			continue
		}
		dir := filepath.Join(root, entry.Name())
		job, ok := byID[entry.Name()]
		if ok && !eligibleOldFinalJob(job, cutoff) {
			report.Skipped++
			continue
		}
		if !ok && !isOldPath(dir, cutoff) {
			report.Skipped++
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			return report, fmt.Errorf("remove job refs %s/%s: %w", token, entry.Name(), err)
		}
		report.JobRefDirsDeleted++
	}
	return report, nil
}

type referenceMeta struct {
	ID        string `json:"id"`
	FileName  string `json:"fileName"`
	CreatedAt string `json:"createdAt"`
}

func readReferenceMeta(path string) (referenceMeta, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return referenceMeta{}, err
	}
	var meta referenceMeta
	if err := json.Unmarshal(data, &meta); err != nil {
		return referenceMeta{}, err
	}
	if strings.TrimSpace(meta.ID) == "" || filepath.Base(meta.FileName) != meta.FileName {
		return referenceMeta{}, fmt.Errorf("invalid upload metadata")
	}
	return meta, nil
}

func addActiveUpload(active map[string]map[string]struct{}, token string, uploadID string) {
	uploadID = strings.TrimSpace(uploadID)
	if uploadID == "" {
		return
	}
	if active[token] == nil {
		active[token] = map[string]struct{}{}
	}
	active[token][uploadID] = struct{}{}
}

func outputKey(token string, date string, fileName string) string {
	return token + "/" + date + "/" + fileName
}

func eligibleOldFinalJob(job jobs.Job, cutoff time.Time) bool {
	if !job.Final() {
		return false
	}
	t := job.FinishedAt
	if t.IsZero() {
		t = job.UpdatedAt
	}
	if t.IsZero() {
		t = job.CreatedAt
	}
	return !t.IsZero() && t.Before(cutoff)
}

func dayStart(t time.Time) time.Time {
	local := t.In(time.Local)
	year, month, day := local.Date()
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local)
}

func isOldFile(path string, cutoff time.Time) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir() && info.ModTime().Before(cutoff)
}

func isOldPath(path string, cutoff time.Time) bool {
	info, err := os.Stat(path)
	return err == nil && info.ModTime().Before(cutoff)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func removeEmptyDir(dir string) bool {
	if err := os.Remove(dir); err == nil {
		return true
	}
	return false
}

func (r *Report) add(next Report) {
	r.OutputFilesDeleted += next.OutputFilesDeleted
	r.OutputDirsDeleted += next.OutputDirsDeleted
	r.UploadsDeleted += next.UploadsDeleted
	r.JobRefDirsDeleted += next.JobRefDirsDeleted
	r.Skipped += next.Skipped
}
