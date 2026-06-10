package store

import (
	"fmt"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"

	"github.com/alexradunet/balaur/internal/models"
)

const settingsKey = "default"

type Models struct {
	app core.App
}

func NewModels(app core.App) *Models {
	return &Models{app: app}
}

func (s *Models) UpsertCatalog(entries []models.CatalogEntry) error {
	for _, entry := range entries {
		rec, err := s.app.FindFirstRecordByData("local_models", "key", entry.Key)
		if err != nil {
			col, colErr := s.app.FindCollectionByNameOrId("local_models")
			if colErr != nil {
				return colErr
			}
			rec = core.NewRecord(col)
			rec.Set("key", entry.Key)
			rec.Set("status", "available")
		}
		rec.Set("name", entry.Name)
		rec.Set("family", entry.Family)
		rec.Set("role", entry.Role)
		rec.Set("release_year", entry.ReleaseYear)
		rec.Set("tool_support", entry.ToolSupport)
		rec.Set("source_page", entry.SourcePage)
		rec.Set("download_url", entry.DownloadURL)
		rec.Set("license", entry.License)
		rec.Set("provenance", entry.Provenance)
		if entry.SizeBytes > 0 && rec.GetInt("size_bytes") == 0 {
			rec.Set("size_bytes", entry.SizeBytes)
		}
		if entry.SHA256 != "" && rec.GetString("sha256") == "" {
			rec.Set("sha256", entry.SHA256)
		}
		if err := s.app.Save(rec); err != nil {
			return err
		}
	}
	return s.ensureSettings()
}

func (s *Models) ListModels() ([]models.LocalModel, error) {
	recs, err := s.app.FindRecordsByFilter("local_models", "", "family,name", 0, 0)
	if err != nil {
		return nil, err
	}
	out := make([]models.LocalModel, 0, len(recs))
	for _, rec := range recs {
		out = append(out, recordToModel(rec))
	}
	return out, nil
}

func (s *Models) ModelByKey(key string) (models.LocalModel, error) {
	rec, err := s.app.FindFirstRecordByData("local_models", "key", key)
	if err != nil {
		return models.LocalModel{}, err
	}
	return recordToModel(rec), nil
}

func (s *Models) ActiveChatModelPath() (string, error) {
	rec, err := s.app.FindFirstRecordByFilter(
		"local_models",
		"active = true && status = 'downloaded' && role = 'chat'",
	)
	if err != nil {
		return "", nil
	}
	return rec.GetString("local_path"), nil
}

func (s *Models) MarkDownloading(key string, total int64) error {
	rec, err := s.recordByKey(key)
	if err != nil {
		return err
	}
	rec.Set("status", "downloading")
	rec.Set("downloaded_bytes", 0)
	rec.Set("error", "")
	if total > 0 {
		rec.Set("size_bytes", total)
	}
	return s.app.Save(rec)
}

func (s *Models) UpdateProgress(key string, downloaded, total int64) error {
	rec, err := s.recordByKey(key)
	if err != nil {
		return err
	}
	rec.Set("downloaded_bytes", downloaded)
	if total > 0 {
		rec.Set("size_bytes", total)
	}
	return s.app.Save(rec)
}

func (s *Models) MarkDownloaded(key, path, sha string, total int64) error {
	rec, err := s.recordByKey(key)
	if err != nil {
		return err
	}
	if err := s.deactivateAll(); err != nil {
		return err
	}
	rec.Set("status", "downloaded")
	rec.Set("local_path", path)
	rec.Set("downloaded_bytes", total)
	rec.Set("size_bytes", total)
	rec.Set("sha256", sha)
	rec.Set("error", "")
	rec.Set("active", true)
	if err := s.app.Save(rec); err != nil {
		return err
	}
	settings, err := s.settings()
	if err != nil {
		return err
	}
	settings.Set("provider", "local")
	settings.Set("active_chat_model", rec.Id)
	return s.app.Save(settings)
}

func (s *Models) MarkFailed(key, message string) error {
	rec, err := s.recordByKey(key)
	if err != nil {
		return err
	}
	rec.Set("status", "failed")
	rec.Set("error", message)
	return s.app.Save(rec)
}

func (s *Models) MarkInterruptedDownloads() error {
	recs, err := s.app.FindRecordsByFilter("local_models", "status = 'downloading'", "", 0, 0)
	if err != nil {
		return err
	}
	for _, rec := range recs {
		rec.Set("status", "failed")
		rec.Set("error", "download interrupted")
		if err := s.app.Save(rec); err != nil {
			return err
		}
	}
	return nil
}

func (s *Models) SelectModel(key string) error {
	rec, err := s.recordByKey(key)
	if err != nil {
		return err
	}
	if rec.GetString("status") != "downloaded" || rec.GetString("local_path") == "" {
		return fmt.Errorf("model is not downloaded")
	}
	if err := s.deactivateAll(); err != nil {
		return err
	}
	rec.Set("active", true)
	if err := s.app.Save(rec); err != nil {
		return err
	}
	settings, err := s.settings()
	if err != nil {
		return err
	}
	settings.Set("provider", "local")
	settings.Set("active_chat_model", rec.Id)
	return s.app.Save(settings)
}

func (s *Models) AuditModel(action, target string, allowed bool, detail map[string]any) {
	col, err := s.app.FindCollectionByNameOrId("audit_log")
	if err != nil {
		return
	}
	rec := core.NewRecord(col)
	rec.Set("actor", "model")
	rec.Set("action", action)
	rec.Set("target", target)
	rec.Set("allowed", allowed)
	if detail != nil {
		rec.Set("detail", detail)
	}
	_ = s.app.Save(rec)
}

func (s *Models) recordByKey(key string) (*core.Record, error) {
	return s.app.FindFirstRecordByData("local_models", "key", key)
}

func (s *Models) deactivateAll() error {
	recs, err := s.app.FindRecordsByFilter("local_models", "active = true", "", 0, 0)
	if err != nil {
		return err
	}
	for _, rec := range recs {
		rec.Set("active", false)
		if err := s.app.Save(rec); err != nil {
			return err
		}
	}
	return nil
}

func (s *Models) ensureSettings() error {
	_, err := s.settings()
	return err
}

func (s *Models) settings() (*core.Record, error) {
	rec, err := s.app.FindFirstRecordByData("model_settings", "key", settingsKey)
	if err == nil {
		return rec, nil
	}
	col, err := s.app.FindCollectionByNameOrId("model_settings")
	if err != nil {
		return nil, err
	}
	rec = core.NewRecord(col)
	rec.Set("key", settingsKey)
	rec.Set("provider", "env")
	if err := s.app.Save(rec); err != nil {
		return nil, err
	}
	return rec, nil
}

func recordToModel(rec *core.Record) models.LocalModel {
	return models.LocalModel{
		ID:              rec.Id,
		Key:             rec.GetString("key"),
		Name:            rec.GetString("name"),
		Family:          rec.GetString("family"),
		Role:            rec.GetString("role"),
		ReleaseYear:     rec.GetInt("release_year"),
		ToolSupport:     rec.GetBool("tool_support"),
		SourcePage:      rec.GetString("source_page"),
		DownloadURL:     rec.GetString("download_url"),
		License:         rec.GetString("license"),
		Provenance:      rec.GetString("provenance"),
		Status:          rec.GetString("status"),
		LocalPath:       rec.GetString("local_path"),
		SizeBytes:       int64(rec.GetInt("size_bytes")),
		DownloadedBytes: int64(rec.GetInt("downloaded_bytes")),
		SHA256:          rec.GetString("sha256"),
		Error:           rec.GetString("error"),
		Active:          rec.GetBool("active"),
	}
}

func CountAudit(app core.App, action string, allowed bool) (int, error) {
	recs, err := app.FindRecordsByFilter(
		"audit_log",
		"action = {:action} && allowed = {:allowed}",
		"", 0, 0,
		dbx.Params{"action": action, "allowed": allowed},
	)
	if err != nil {
		return 0, err
	}
	return len(recs), nil
}
