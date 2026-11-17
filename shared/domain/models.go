package domain

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
)

// Schedule represents a recurring announcement rule evaluated by the shared scheduler.
type Schedule struct {
	ID              string     `json:"id"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	AudioID         string     `json:"audioId"`
	TimeOfDay       string     `json:"timeOfDay"` // Format "HH:MM" (24-jam)
	Days            []int      `json:"days"`      // 0=Minggu .. 6=Sabtu; kosong = setiap hari
	Enabled         bool       `json:"enabled"`
	Volume          int        `json:"volume"` // 0..100
	LastTriggeredAt *time.Time `json:"lastTriggeredAt,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

// AudioAsset describes a validated local audio file managed by the application.
type AudioAsset struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	FileName  string    `json:"fileName"`
	FilePath  string    `json:"filePath"`
	Format    string    `json:"format"` // "mp3" atau "wav"
	SizeBytes int64     `json:"sizeBytes"`
	AddedAt   time.Time `json:"addedAt"`
}

// AppConfig holds persistent user preferences across desktop and mobile.
type AppConfig struct {
	StartMinimized   bool   `json:"startMinimized"`
	AutostartEnabled bool   `json:"autostartEnabled"`
	DefaultVolume    int    `json:"defaultVolume"`
	SchedulerEnabled bool   `json:"schedulerEnabled"`
	Language         string `json:"language"`
}

// DefaultAppConfig returns safe initial application settings in Bahasa Indonesia.
func DefaultAppConfig() AppConfig {
	return AppConfig{
		StartMinimized:   true,
		AutostartEnabled: false,
		DefaultVolume:    85,
		SchedulerEnabled: true,
		Language:         "id",
	}
}

// LogEntry records an operational event displayed in the UI diagnostics log.
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Source    string    `json:"source"`
	Message   string    `json:"message"`
}

// PlatformCapabilities reports which OS-specific integrations are active at runtime.
type PlatformCapabilities struct {
	Platform                    string   `json:"platform"`
	IsMobile                    bool     `json:"isMobile"`
	IsDesktop                   bool     `json:"isDesktop"`
	SupportsAutostart           bool     `json:"supportsAutostart"`
	SupportsSystemTray          bool     `json:"supportsSystemTray"`
	SupportsBackgroundScheduler bool     `json:"supportsBackgroundScheduler"`
	SupportsNativeFilePicker    bool     `json:"supportsNativeFilePicker"`
	SupportedAudioFormats       []string `json:"supportedAudioFormats"`
}

// NextAnnouncementInfo summarizes the nearest upcoming enabled schedule.
type NextAnnouncementInfo struct {
	ScheduleID   string    `json:"scheduleId"`
	Title        string    `json:"title"`
	AudioID      string    `json:"audioId"`
	AudioName    string    `json:"audioName"`
	ScheduledAt  time.Time `json:"scheduledAt"`
	RemainingSec int64     `json:"remainingSec"`
}

// PlaybackState tracks active audio output status.
type PlaybackState struct {
	IsPlaying  bool       `json:"isPlaying"`
	AudioID    string     `json:"audioId,omitempty"`
	AudioName  string     `json:"audioName,omitempty"`
	ScheduleID string     `json:"scheduleId,omitempty"`
	StartedAt  *time.Time `json:"startedAt,omitempty"`
}

// DashboardState aggregates runtime status for the UI dashboard and tray updates.
type DashboardState struct {
	Platform            PlatformCapabilities  `json:"platform"`
	SchedulerRunning    bool                  `json:"schedulerRunning"`
	AutostartEnabled    bool                  `json:"autostartEnabled"`
	ActiveScheduleCount int                   `json:"activeScheduleCount"`
	TotalScheduleCount  int                   `json:"totalScheduleCount"`
	TotalAudioCount     int                   `json:"totalAudioCount"`
	NextAnnouncement    *NextAnnouncementInfo `json:"nextAnnouncement,omitempty"`
	Playback            PlaybackState         `json:"playback"`
}

// ParseTimeOfDay parses "HH:MM" into hour and minute values.
func ParseTimeOfDay(hhmm string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(hhmm), ":")
	if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 2 {
		return 0, 0, errors.New("format waktu harus HH:MM (contoh 07:30)")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, errors.New("jam harus di antara 00 dan 23")
	}
	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return 0, 0, errors.New("menit harus di antara 00 dan 59")
	}
	return hour, minute, nil
}

// ValidateSchedule checks schedule fields and returns an Indonesian error message if invalid.
func ValidateSchedule(s Schedule) error {
	if strings.TrimSpace(s.Title) == "" {
		return errors.New("judul pengumuman wajib diisi")
	}
	if strings.TrimSpace(s.AudioID) == "" {
		return errors.New("berkas audio wajib dipilih")
	}
	if _, _, err := ParseTimeOfDay(s.TimeOfDay); err != nil {
		return err
	}
	for _, day := range s.Days {
		if day < 0 || day > 6 {
			return fmt.Errorf("kode hari tidak valid: %d (harus 0-6)", day)
		}
	}
	if s.Volume < 0 || s.Volume > 100 {
		return errors.New("volume harus berada di antara 0 dan 100")
	}
	return nil
}

// NormalizeAudioFormat validates supported audio extensions (.mp3, .mpeg, .mpga, .wav) without FFmpeg.
func NormalizeAudioFormat(pathOrName string) (string, error) {
	ext := strings.ToLower(strings.TrimSpace(filepath.Ext(pathOrName)))
	switch ext {
	case ".mp3":
		return "mp3", nil
	case ".mpeg":
		return "mpeg", nil
	case ".mpga":
		return "mpga", nil
	case ".wav":
		return "wav", nil
	case ".mp1", ".mp2", ".mpa", ".m4a", ".aac", ".ogg", ".flac":
		return "", fmt.Errorf("format %s belum didukung tanpa FFmpeg; gunakan berkas .mp3, .mpeg (MPEG Layer III), atau .wav", ext)
	default:
		return "", fmt.Errorf("format berkas audio %q tidak didukung; gunakan .mp3, .mpeg, atau .wav", ext)
	}
}

// MatchesWeekday returns true if the schedule applies on the given weekday.
func (s Schedule) MatchesWeekday(wd time.Weekday) bool {
	if len(s.Days) == 0 {
		return true
	}
	return slices.Contains(s.Days, int(wd))
}
