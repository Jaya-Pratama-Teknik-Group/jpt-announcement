package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
)

func TestStoreConfigSchedulesAndAudio(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir)
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}

	// Default config
	cfg, err := store.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.Language != "id" || cfg.DefaultVolume != 85 {
		t.Fatalf("unexpected default config: %+v", cfg)
	}

	cfg.AutostartEnabled = true
	cfg.DefaultVolume = 90
	if err := store.SaveConfig(cfg); err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	reloaded, err := store.LoadConfig()
	if err != nil || !reloaded.AutostartEnabled || reloaded.DefaultVolume != 90 {
		t.Fatalf("config did not persist properly: %+v, err=%v", reloaded, err)
	}

	// Audio import via bytes and file (.mp3 and .mpeg)
	asset, err := store.ImportAudioBytes("bel-masuk.mp3", "Bel Masuk", []byte("ID3mockdata"))
	if err != nil {
		t.Fatalf("ImportAudioBytes failed: %v", err)
	}
	if _, err := os.Stat(asset.FilePath); err != nil {
		t.Fatalf("expected managed audio file at %s: %v", asset.FilePath, err)
	}

	mpegAsset, err := store.ImportAudioBytes("pengumuman.mpeg", "Pengumuman MPEG", []byte("ID3mockmpeg"))
	if err != nil {
		t.Fatalf("ImportAudioBytes (.mpeg) failed: %v", err)
	}
	if mpegAsset.Format != "mpeg" {
		t.Fatalf("expected format mpeg, got %q", mpegAsset.Format)
	}
	if _, err := os.Stat(mpegAsset.FilePath); err != nil {
		t.Fatalf("expected managed mpeg file at %s: %v", mpegAsset.FilePath, err)
	}

	// Unsupported format rejection
	if _, err := store.ImportAudioBytes("suara.m4a", "M4A", []byte("data")); err == nil {
		t.Fatal("expected error for unsupported .m4a format")
	}

	// Save schedule referencing asset
	sch, err := store.SaveSchedule(domain.Schedule{
		Title:     "Bel Pagi",
		AudioID:   asset.ID,
		TimeOfDay: "07:00",
		Days:      []int{1, 2, 3, 4, 5},
		Enabled:   true,
		Volume:    80,
	})
	if err != nil {
		t.Fatalf("SaveSchedule failed: %v", err)
	}

	// Deleting audio referenced by schedule must fail
	if err := store.DeleteAudio(asset.ID); err == nil {
		t.Fatal("expected DeleteAudio to fail while referenced by schedule")
	}

	// Delete schedule then delete audio
	if err := store.DeleteSchedule(sch.ID); err != nil {
		t.Fatalf("DeleteSchedule failed: %v", err)
	}
	if err := store.DeleteAudio(asset.ID); err != nil {
		t.Fatalf("DeleteAudio failed after schedule removal: %v", err)
	}

	// Verify file removed from managed folder
	if _, err := os.Stat(filepath.Join(dir, "audio", asset.ID+".mp3")); !os.IsNotExist(err) {
		t.Fatalf("expected managed audio file to be deleted, stat err=%v", err)
	}
}
