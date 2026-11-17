package announcement

import (
	"context"
	"encoding/base64"
	"sync"
	"testing"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/config"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/logging"
)

type fakeAudio struct {
	mu      sync.Mutex
	playing bool
	last    string
}

func (f *fakeAudio) Play(_ context.Context, filePath string, _ int, onFinish func(error)) error {
	f.mu.Lock()
	f.playing = true
	f.last = filePath
	f.mu.Unlock()
	if onFinish != nil {
		onFinish(nil)
	}
	f.mu.Lock()
	f.playing = false
	f.mu.Unlock()
	return nil
}

func (f *fakeAudio) Stop() error {
	f.mu.Lock()
	f.playing = false
	f.mu.Unlock()
	return nil
}

func (f *fakeAudio) IsPlaying() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.playing
}

type fakeStartup struct {
	enabled bool
}

func (f *fakeStartup) Enable() error          { f.enabled = true; return nil }
func (f *fakeStartup) Disable() error         { f.enabled = false; return nil }
func (f *fakeStartup) IsEnabled() (bool, error) { return f.enabled, nil }

func TestServiceLifecycleAndAutostart(t *testing.T) {
	store, err := config.NewStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewStore failed: %v", err)
	}
	audio := &fakeAudio{}
	startup := &fakeStartup{}

	svc := NewService(Options{
		Store:   store,
		Logger:  logging.New(""),
		Audio:   audio,
		Startup: startup,
		Capabilities: domain.PlatformCapabilities{
			Platform:          "windows",
			IsDesktop:         true,
			SupportsAutostart: true,
		},
	})

	svc.Start()
	defer svc.Shutdown()

	enabled, err := svc.SetAutostartEnabled(true)
	if err != nil || !enabled || !startup.enabled {
		t.Fatalf("expected autostart enabled, got enabled=%v err=%v", enabled, err)
	}

	b64 := base64.StdEncoding.EncodeToString([]byte("RIFFmockwav"))
	asset, err := svc.ImportAudioBase64("bel.wav", "Bel Sekolah", b64)
	if err != nil {
		t.Fatalf("ImportAudioBase64 failed: %v", err)
	}

	sch, err := svc.SaveSchedule(domain.Schedule{
		Title:     "Bel Istirahat",
		AudioID:   asset.ID,
		TimeOfDay: "10:00",
		Enabled:   true,
		Volume:    90,
	})
	if err != nil {
		t.Fatalf("SaveSchedule failed: %v", err)
	}

	if err := svc.PlayScheduleNow(sch.ID); err != nil {
		t.Fatalf("PlayScheduleNow failed: %v", err)
	}
	if audio.last != asset.FilePath {
		t.Fatalf("expected audio path %s, got %s", asset.FilePath, audio.last)
	}

	state, err := svc.GetDashboardState()
	if err != nil {
		t.Fatalf("GetDashboardState failed: %v", err)
	}
	if state.ActiveScheduleCount != 1 || state.TotalAudioCount != 1 {
		t.Fatalf("unexpected dashboard state: %+v", state)
	}
}
