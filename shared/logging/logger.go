package logging

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/shared/domain"
)

const defaultCapacity = 200

// Logger stores structured operational logs in memory and optionally appends to a log file.
type Logger struct {
	mu       sync.RWMutex
	entries  []domain.LogEntry
	capacity int
	filePath string
}

// New creates a Logger with an optional file destination.
func New(logFilePath string) *Logger {
	if logFilePath != "" {
		_ = os.MkdirAll(filepath.Dir(logFilePath), 0o755)
	}
	return &Logger{
		entries:  make([]domain.LogEntry, 0, defaultCapacity),
		capacity: defaultCapacity,
		filePath: logFilePath,
	}
}

// Info records an informational entry.
func (l *Logger) Info(source, message string) {
	l.add("INFO", source, message)
}

// Warn records a warning entry.
func (l *Logger) Warn(source, message string) {
	l.add("WARN", source, message)
}

// Error records an error entry.
func (l *Logger) Error(source, message string) {
	l.add("ERROR", source, message)
}

func (l *Logger) add(level, source, message string) {
	entry := domain.LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Source:    source,
		Message:   message,
	}

	l.mu.Lock()
	if len(l.entries) >= l.capacity {
		copy(l.entries, l.entries[1:])
		l.entries[len(l.entries)-1] = entry
	} else {
		l.entries = append(l.entries, entry)
	}
	filePath := l.filePath
	l.mu.Unlock()

	if filePath != "" {
		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = fmt.Fprintf(f, "%s [%s] %s: %s\n", entry.Timestamp.Format(time.RFC3339), entry.Level, entry.Source, entry.Message)
			_ = f.Close()
		}
	}
}

// List returns logs ordered newest-first.
func (l *Logger) List() []domain.LogEntry {
	l.mu.RLock()
	defer l.mu.RUnlock()

	out := make([]domain.LogEntry, len(l.entries))
	for i := range l.entries {
		out[i] = l.entries[len(l.entries)-1-i]
	}
	return out
}
