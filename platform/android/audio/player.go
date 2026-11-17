//go:build android

package audio

import (
	"github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform"
	sharedaudio "github.com/Jaya-Pratama-Teknik-Group/jpt-announcement/platform/audio"
)

// NewPlayer returns the Android audio playback implementation (Oboe via oto/v3 and beep/v2).
func NewPlayer() platform.AudioPlayer {
	return sharedaudio.NewBeepPlayer()
}
