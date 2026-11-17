//go:build windows

package audio

import (
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	sharedaudio "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/audio"
)

// NewPlayer returns the Windows audio playback implementation backed by gopxl/beep/v2 (pure Go).
func NewPlayer() platform.AudioPlayer {
	return sharedaudio.NewBeepPlayer()
}
