package config

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
)

// Store manages persistent JSON configuration, schedules, and managed local audio files.
type Store struct {
	mu       sync.RWMutex
	baseDir  string
	audioDir string
}

// NewStore initializes the store directory under baseDir (or UserConfigDir/JPTAnnounce if empty).
func NewStore(baseDir string) (*Store, error) {
	if strings.TrimSpace(baseDir) == "" {
		userCfg, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("gagal menentukan direktori konfigurasi: %w", err)
		}
		baseDir = filepath.Join(userCfg, "JPTAnnounce")
	}
	audioDir := filepath.Join(baseDir, "audio")
	if err := os.MkdirAll(audioDir, 0o755); err != nil {
		return nil, fmt.Errorf("gagal membuat direktori audio: %w", err)
	}
	return &Store{
		baseDir:  baseDir,
		audioDir: audioDir,
	}, nil
}

// BaseDir returns the root storage path for configs and logs.
func (s *Store) BaseDir() string {
	return s.baseDir
}

// LoadConfig reads AppConfig or returns defaults when not yet created.
func (s *Store) LoadConfig() (domain.AppConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	cfg := domain.DefaultAppConfig()
	path := filepath.Join(s.baseDir, "config.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("gagal membaca pengaturan: %w", err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("berkas pengaturan rusak: %w", err)
	}
	if cfg.Language == "" {
		cfg.Language = "id"
	}
	if cfg.DefaultVolume < 0 || cfg.DefaultVolume > 100 {
		cfg.DefaultVolume = 85
	}
	if cfg.DuckVolumePercent < 0 || cfg.DuckVolumePercent > 100 {
		cfg.DuckVolumePercent = 25
	}
	return cfg, nil
}

// SaveConfig persists AppConfig atomically.
func (s *Store) SaveConfig(cfg domain.AppConfig) error {
	if cfg.DefaultVolume < 0 || cfg.DefaultVolume > 100 {
		return errors.New("volume bawaan harus di antara 0 dan 100")
	}
	if cfg.DuckVolumePercent < 0 || cfg.DuckVolumePercent > 100 {
		return errors.New("persentase volume aplikasi lain harus di antara 0 dan 100")
	}
	if cfg.Language == "" {
		cfg.Language = "id"
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return writeJSONAtomic(filepath.Join(s.baseDir, "config.json"), cfg)
}

// ListSchedules returns all saved announcement schedules.
func (s *Store) ListSchedules() ([]domain.Schedule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadSchedulesLocked()
}

// SaveSchedule inserts or updates a schedule.
func (s *Store) SaveSchedule(item domain.Schedule) (domain.Schedule, error) {
	times, err := domain.NormalizeScheduleTimes(item.TimeOfDay, item.TimesOfDay)
	if err != nil {
		return domain.Schedule{}, err
	}
	item.TimesOfDay = times
	item.TimeOfDay = times[0]
	if err := domain.ValidateSchedule(item); err != nil {
		return domain.Schedule{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadSchedulesLocked()
	if err != nil {
		return domain.Schedule{}, err
	}

	now := time.Now()
	item.Title = strings.TrimSpace(item.Title)
	item.Description = strings.TrimSpace(item.Description)
	slices.Sort(item.Days)
	item.Days = slices.Compact(item.Days)

	if item.ID == "" {
		item.ID = newID("sch")
		item.CreatedAt = now
		item.UpdatedAt = now
		items = append(items, item)
	} else {
		updated := false
		for i := range items {
			if items[i].ID == item.ID {
				item.CreatedAt = items[i].CreatedAt
				if item.LastTriggeredAt == nil {
					item.LastTriggeredAt = items[i].LastTriggeredAt
				}
				item.UpdatedAt = now
				items[i] = item
				updated = true
				break
			}
		}
		if !updated {
			item.CreatedAt = now
			item.UpdatedAt = now
			items = append(items, item)
		}
	}

	if err := writeJSONAtomic(filepath.Join(s.baseDir, "schedules.json"), items); err != nil {
		return domain.Schedule{}, err
	}
	return item, nil
}

// DeleteSchedule removes a schedule by ID.
func (s *Store) DeleteSchedule(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadSchedulesLocked()
	if err != nil {
		return err
	}
	filtered := make([]domain.Schedule, 0, len(items))
	found := false
	for _, item := range items {
		if item.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return fmt.Errorf("jadwal dengan ID %q tidak ditemukan", id)
	}
	return writeJSONAtomic(filepath.Join(s.baseDir, "schedules.json"), filtered)
}

// MarkTriggered updates LastTriggeredAt for a schedule.
func (s *Store) MarkTriggered(id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadSchedulesLocked()
	if err != nil {
		return err
	}
	for i := range items {
		if items[i].ID == id {
			t := at
			items[i].LastTriggeredAt = &t
			items[i].UpdatedAt = time.Now()
			return writeJSONAtomic(filepath.Join(s.baseDir, "schedules.json"), items)
		}
	}
	return nil
}

// ListAudio returns all registered audio assets.
func (s *Store) ListAudio() ([]domain.AudioAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loadAudioLocked()
}

// GetAudio returns a single audio asset by ID.
func (s *Store) GetAudio(id string) (domain.AudioAsset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items, err := s.loadAudioLocked()
	if err != nil {
		return domain.AudioAsset{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return domain.AudioAsset{}, fmt.Errorf("berkas audio dengan ID %q tidak ditemukan", id)
}

// ImportAudioFile copies a local audio file into app-managed storage and registers it.
func (s *Store) ImportAudioFile(sourcePath, displayName string) (domain.AudioAsset, error) {
	format, err := domain.NormalizeAudioFormat(sourcePath)
	if err != nil {
		return domain.AudioAsset{}, err
	}

	info, err := os.Stat(sourcePath)
	if err != nil {
		return domain.AudioAsset{}, fmt.Errorf("berkas audio tidak dapat diakses: %w", err)
	}
	if info.IsDir() {
		return domain.AudioAsset{}, errors.New("path yang dipilih bukan berkas audio")
	}

	id := newID("aud")
	origName := filepath.Base(sourcePath)
	destName := id + "." + format
	destPath := filepath.Join(s.audioDir, destName)

	if err := copyFile(sourcePath, destPath); err != nil {
		return domain.AudioAsset{}, fmt.Errorf("gagal menyalin berkas audio: %w", err)
	}

	return s.registerAudioLocked(id, origName, destPath, format, info.Size(), displayName)
}

// ImportAudioBytes writes uploaded bytes (e.g. from mobile WebView file input) into managed storage.
func (s *Store) ImportAudioBytes(fileName, displayName string, data []byte) (domain.AudioAsset, error) {
	if len(data) == 0 {
		return domain.AudioAsset{}, errors.New("berkas audio kosong")
	}
	format, err := domain.NormalizeAudioFormat(fileName)
	if err != nil {
		return domain.AudioAsset{}, err
	}

	id := newID("aud")
	destPath := filepath.Join(s.audioDir, id+"."+format)
	if err := os.WriteFile(destPath, data, 0o644); err != nil {
		return domain.AudioAsset{}, fmt.Errorf("gagal menyimpan berkas audio: %w", err)
	}

	return s.registerAudioLocked(id, filepath.Base(fileName), destPath, format, int64(len(data)), displayName)
}

// DeleteAudio removes an audio entry and its managed file when not referenced by schedules.
func (s *Store) DeleteAudio(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	schedules, err := s.loadSchedulesLocked()
	if err != nil {
		return err
	}
	for _, sch := range schedules {
		if sch.AudioID == id {
			return fmt.Errorf("audio masih dipakai oleh jadwal %q", sch.Title)
		}
	}

	items, err := s.loadAudioLocked()
	if err != nil {
		return err
	}
	filtered := make([]domain.AudioAsset, 0, len(items))
	var targetPath string
	found := false
	for _, item := range items {
		if item.ID == id {
			found = true
			targetPath = item.FilePath
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return fmt.Errorf("berkas audio dengan ID %q tidak ditemukan", id)
	}
	if err := writeJSONAtomic(filepath.Join(s.baseDir, "audio.json"), filtered); err != nil {
		return err
	}
	if targetPath != "" {
		_ = os.Remove(targetPath)
	}
	return nil
}

func (s *Store) registerAudioLocked(id, origName, destPath, format string, size int64, displayName string) (domain.AudioAsset, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	items, err := s.loadAudioLocked()
	if err != nil {
		return domain.AudioAsset{}, err
	}

	name := strings.TrimSpace(displayName)
	if name == "" {
		name = strings.TrimSuffix(origName, filepath.Ext(origName))
	}

	asset := domain.AudioAsset{
		ID:        id,
		Name:      name,
		FileName:  origName,
		FilePath:  destPath,
		Format:    format,
		SizeBytes: size,
		AddedAt:   time.Now(),
	}
	items = append(items, asset)
	if err := writeJSONAtomic(filepath.Join(s.baseDir, "audio.json"), items); err != nil {
		return domain.AudioAsset{}, err
	}
	return asset, nil
}

func (s *Store) loadSchedulesLocked() ([]domain.Schedule, error) {
	path := filepath.Join(s.baseDir, "schedules.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []domain.Schedule{}, nil
		}
		return nil, fmt.Errorf("gagal membaca jadwal: %w", err)
	}
	var items []domain.Schedule
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("berkas jadwal rusak: %w", err)
	}
	for i := range items {
		if times, err := domain.NormalizeScheduleTimes(items[i].TimeOfDay, items[i].TimesOfDay); err == nil {
			items[i].TimesOfDay = times
			items[i].TimeOfDay = times[0]
		}
	}
	return items, nil
}

func (s *Store) loadAudioLocked() ([]domain.AudioAsset, error) {
	path := filepath.Join(s.baseDir, "audio.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []domain.AudioAsset{}, nil
		}
		return nil, fmt.Errorf("gagal membaca daftar audio: %w", err)
	}
	var items []domain.AudioAsset
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("berkas daftar audio rusak: %w", err)
	}
	return items, nil
}

func writeJSONAtomic(target string, payload any) error {
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, target)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func newID(prefix string) string {
	var buf [6]byte
	_, _ = rand.Read(buf[:])
	return prefix + "_" + hex.EncodeToString(buf[:])
}
