package scheduler

import (
	"sync"
	"testing"
	"time"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/logging"
)

type memoryProvider struct {
	mu        sync.Mutex
	schedules []domain.Schedule
}

func (m *memoryProvider) ListSchedules() ([]domain.Schedule, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]domain.Schedule, len(m.schedules))
	copy(out, m.schedules)
	return out, nil
}

func (m *memoryProvider) MarkTriggered(id string, at time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.schedules {
		if m.schedules[i].ID == id {
			t := at
			m.schedules[i].LastTriggeredAt = &t
		}
	}
	return nil
}

func TestEvaluateTriggersOncePerMinuteAndRespectsWeekday(t *testing.T) {
	loc := time.FixedZone("WIB", 7*3600)
	// Tuesday, 2026-11-17 07:30:05
	now := time.Date(2026, 11, 17, 7, 30, 5, 0, loc)

	provider := &memoryProvider{
		schedules: []domain.Schedule{
			{
				ID:        "s1",
				Title:     "Apel Pagi",
				AudioID:   "a1",
				TimeOfDay: "07:30",
				Days:      []int{int(time.Tuesday)},
				Enabled:   true,
				Volume:    80,
			},
			{
				ID:        "s2",
				Title:     "Khusus Rabu",
				AudioID:   "a1",
				TimeOfDay: "07:30",
				Days:      []int{int(time.Wednesday)},
				Enabled:   true,
				Volume:    80,
			},
			{
				ID:        "s3",
				Title:     "Nonaktif",
				AudioID:   "a1",
				TimeOfDay: "07:30",
				Days:      []int{int(time.Tuesday)},
				Enabled:   false,
				Volume:    80,
			},
		},
	}

	var triggered []string
	sched := New(provider, logging.New(""), func(s domain.Schedule) error {
		triggered = append(triggered, s.ID)
		return nil
	})

	sched.Evaluate(now)
	// Same minute 20 seconds later should NOT re-trigger
	sched.Evaluate(now.Add(20 * time.Second))

	if len(triggered) != 1 || triggered[0] != "s1" {
		t.Fatalf("expected only [s1] to trigger once, got %v", triggered)
	}
}

func TestFindNextAnnouncement(t *testing.T) {
	loc := time.FixedZone("WIB", 7*3600)
	now := time.Date(2026, 11, 17, 8, 0, 0, 0, loc) // Tuesday

	schedules := []domain.Schedule{
		{ID: "s1", Title: "Pagi", AudioID: "a1", TimeOfDay: "07:30", Enabled: true},  // tomorrow 07:30
		{ID: "s2", Title: "Siang", AudioID: "a2", TimeOfDay: "12:00", Enabled: true}, // today 12:00
		{ID: "s3", Title: "Off", AudioID: "a1", TimeOfDay: "09:00", Enabled: false},
	}

	next := FindNextAnnouncement(schedules, map[string]string{"a1": "Bel Pagi", "a2": "Bel Siang"}, now)
	if next == nil {
		t.Fatal("expected next announcement, got nil")
	}
	if next.ScheduleID != "s2" || next.AudioName != "Bel Siang" {
		t.Fatalf("expected s2 (Bel Siang), got %+v", next)
	}
	if next.RemainingSec != 4*3600 {
		t.Fatalf("expected 14400 remaining seconds, got %d", next.RemainingSec)
	}
}

func TestEvaluateAndNextOccurrenceMultipleTimesPerSchedule(t *testing.T) {
	loc := time.FixedZone("WIB", 7*3600)
	provider := &memoryProvider{
		schedules: []domain.Schedule{
			{
				ID:         "multi",
				Title:      "Pengumuman Shift",
				AudioID:    "a1",
				TimeOfDay:  "07:30",
				TimesOfDay: []string{"07:30", "12:00", "16:30"},
				Enabled:    true,
				Volume:     85,
			},
		},
	}

	var triggered []string
	sched := New(provider, logging.New(""), func(s domain.Schedule) error {
		triggered = append(triggered, s.ID)
		return nil
	})

	// 07:30 triggers once
	tMorning := time.Date(2026, 11, 17, 7, 30, 0, 0, loc)
	sched.Evaluate(tMorning)
	sched.Evaluate(tMorning.Add(15 * time.Second))
	if len(triggered) != 1 {
		t.Fatalf("expected 1 trigger at 07:30, got %d", len(triggered))
	}

	// Next occurrence from 08:00 should be 12:00 same day
	next, ok := NextOccurrence(provider.schedules[0], time.Date(2026, 11, 17, 8, 0, 0, 0, loc))
	if !ok || next.Hour() != 12 || next.Minute() != 0 {
		t.Fatalf("expected next occurrence at 12:00, got %v (ok=%v)", next, ok)
	}

	// 12:00 triggers second time for the same schedule
	sched.Evaluate(time.Date(2026, 11, 17, 12, 0, 10, 0, loc))
	if len(triggered) != 2 {
		t.Fatalf("expected 2 triggers after 12:00, got %d", len(triggered))
	}

	// 16:30 triggers third time for the same schedule
	sched.Evaluate(time.Date(2026, 11, 17, 16, 30, 5, 0, loc))
	if len(triggered) != 3 {
		t.Fatalf("expected 3 triggers after 16:30, got %d", len(triggered))
	}
}
