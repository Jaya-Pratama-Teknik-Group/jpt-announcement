package audio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"
)

type bytesReadSeekCloser struct {
	*bytes.Reader
}

func (bytesReadSeekCloser) Close() error { return nil }

// prepareMPEGAudioStream inspects an MPEG audio file. Raw MPEG Layer III streams
// (including .mpeg/.mpga files with audio/mpeg payload) are returned as-is for
// streaming; MPEG Program Stream containers (0x000001BA) have their MPEG audio
// elementary stream packets (0xC0-0xDF) demuxed into a seekable stream.
func prepareMPEGAudioStream(f *os.File) (io.ReadCloser, error) {
	var magic [4]byte
	n, err := io.ReadFull(f, magic[:])
	if _, seekErr := f.Seek(0, io.SeekStart); seekErr != nil {
		return nil, seekErr
	}
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return nil, err
	}
	if n < 4 || magic[0] != 0x00 || magic[1] != 0x00 || magic[2] != 0x01 || magic[3] != 0xBA {
		return f, nil
	}

	raw, err := io.ReadAll(f)
	_ = f.Close()
	if err != nil {
		return nil, err
	}
	extracted, err := extractMPEGPSAudio(raw)
	if err != nil {
		return nil, err
	}
	return bytesReadSeekCloser{Reader: bytes.NewReader(extracted)}, nil
}

// extractMPEGPSAudio extracts MPEG audio elementary stream bytes (stream IDs 0xC0-0xDF)
// from an MPEG-1/MPEG-2 Program Stream container.
func extractMPEGPSAudio(data []byte) ([]byte, error) {
	var out bytes.Buffer
	i := 0
	for i+4 <= len(data) {
		if data[i] != 0x00 || data[i+1] != 0x00 || data[i+2] != 0x01 {
			i++
			continue
		}
		streamID := data[i+3]
		switch {
		case streamID == 0xBA:
			// Pack header: MPEG-2 if top two bits of byte 4 are '01', else MPEG-1.
			if i+5 > len(data) {
				i = len(data)
				continue
			}
			if (data[i+4] & 0xC0) == 0x40 {
				if i+14 > len(data) {
					i = len(data)
					continue
				}
				stuffing := int(data[i+13] & 0x07)
				i += 14 + stuffing
			} else {
				i += 12
			}
		case streamID == 0xB9:
			// MPEG Program End Code
			i = len(data)
		case streamID >= 0xBB && streamID <= 0xEF:
			if i+6 > len(data) {
				i = len(data)
				continue
			}
			packetLen := int(binary.BigEndian.Uint16(data[i+4 : i+6]))
			payloadStart := i + 6
			packetEnd := payloadStart + packetLen
			if packetLen == 0 || packetEnd > len(data) {
				packetEnd = len(data)
			}
			// Audio elementary streams use PES stream IDs 0xC0..0xDF.
			if streamID >= 0xC0 && streamID <= 0xDF && payloadStart < packetEnd {
				audioStart := skipPESHeader(data[payloadStart:packetEnd])
				if payloadStart+audioStart < packetEnd {
					out.Write(data[payloadStart+audioStart : packetEnd])
				}
			}
			i = packetEnd
		default:
			i += 4
		}
	}
	if out.Len() == 0 {
		return nil, errors.New("berkas wadah MPEG tidak memiliki aliran audio yang dapat diputar")
	}
	return out.Bytes(), nil
}

func skipPESHeader(pes []byte) int {
	if len(pes) == 0 {
		return 0
	}
	// MPEG-2 PES header starts with '10' in bits 7..6.
	if (pes[0] & 0xC0) == 0x80 {
		if len(pes) < 3 {
			return len(pes)
		}
		headerLen := int(pes[2])
		offset := 3 + headerLen
		if offset > len(pes) {
			return len(pes)
		}
		return offset
	}
	// MPEG-1 PES header: skip stuffing 0xFF bytes, optional STD buffer scale/size, and PTS/DTS.
	idx := 0
	for idx < len(pes) && pes[idx] == 0xFF {
		idx++
	}
	if idx < len(pes) && (pes[idx]&0xC0) == 0x40 {
		idx += 2
	}
	if idx >= len(pes) {
		return len(pes)
	}
	switch pes[idx] & 0xF0 {
	case 0x20:
		idx += 5
	case 0x30:
		idx += 10
	default:
		idx++
	}
	if idx > len(pes) {
		return len(pes)
	}
	return idx
}
