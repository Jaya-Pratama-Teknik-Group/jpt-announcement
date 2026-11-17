package audio

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestExtractMPEGPSAudio(t *testing.T) {
	var ps bytes.Buffer
	// MPEG-2 Pack Header (14 bytes, 0 stuffing)
	ps.Write([]byte{0x00, 0x00, 0x01, 0xBA, 0x44, 0x00, 0x04, 0x00, 0x04, 0x01, 0x00, 0x00, 0x03, 0xF8})

	// Video PES packet (0xE0) that must be ignored
	videoPayload := []byte{0x80, 0x00, 0x00, 0xAA, 0xBB}
	ps.Write([]byte{0x00, 0x00, 0x01, 0xE0})
	_ = binary.Write(&ps, binary.BigEndian, uint16(len(videoPayload)))
	ps.Write(videoPayload)

	// Audio PES packet (0xC0) with 3-byte MPEG-2 PES header + payload
	audioData := []byte{0xFF, 0xFB, 0x90, 0x64, 0x01, 0x02}
	pesHeader := []byte{0x80, 0x80, 0x01, 0x00}
	ps.Write([]byte{0x00, 0x00, 0x01, 0xC0})
	_ = binary.Write(&ps, binary.BigEndian, uint16(len(pesHeader)+len(audioData)))
	ps.Write(pesHeader)
	ps.Write(audioData)

	// Program End Code
	ps.Write([]byte{0x00, 0x00, 0x01, 0xB9})

	got, err := extractMPEGPSAudio(ps.Bytes())
	if err != nil {
		t.Fatalf("extractMPEGPSAudio failed: %v", err)
	}
	if !bytes.Equal(got, audioData) {
		t.Fatalf("expected %x, got %x", audioData, got)
	}
}

func TestPrepareMPEGAudioStreamPassthrough(t *testing.T) {
	tmp := filepath.Join(t.TempDir(), "sample.mpeg")
	payload := []byte("ID3\x03\x00\x00\x00\x00\x00\x00mockmp3")
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}
	f, err := os.Open(tmp)
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	rc, err := prepareMPEGAudioStream(f)
	if err != nil {
		t.Fatalf("prepareMPEGAudioStream failed: %v", err)
	}
	defer rc.Close()

	got, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if !bytes.Equal(got, payload) {
		t.Fatalf("expected passthrough payload %q, got %q", payload, got)
	}
}
