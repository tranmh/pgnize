package tts

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestPCMToWAVHeader(t *testing.T) {
	pcm := []byte{0x01, 0x02, 0x03, 0x04} // 2 samples, 16-bit mono
	wav := pcmToWAV(pcm, 24000, 16, 1)

	if len(wav) != 44+len(pcm) {
		t.Fatalf("wav length = %d, want %d (44-byte header + %d data)", len(wav), 44+len(pcm), len(pcm))
	}
	if !bytes.Equal(wav[0:4], []byte("RIFF")) {
		t.Errorf("missing RIFF magic, got %q", wav[0:4])
	}
	if !bytes.Equal(wav[8:12], []byte("WAVE")) {
		t.Errorf("missing WAVE magic, got %q", wav[8:12])
	}
	if !bytes.Equal(wav[12:16], []byte("fmt ")) {
		t.Errorf("missing fmt chunk, got %q", wav[12:16])
	}
	if !bytes.Equal(wav[36:40], []byte("data")) {
		t.Errorf("missing data chunk, got %q", wav[36:40])
	}

	// RIFF chunk size = 36 + dataLen.
	if got := binary.LittleEndian.Uint32(wav[4:8]); got != uint32(36+len(pcm)) {
		t.Errorf("riff size = %d, want %d", got, 36+len(pcm))
	}
	// PCM format (1), 1 channel, 24000 Hz, 16 bits.
	if got := binary.LittleEndian.Uint16(wav[20:22]); got != 1 {
		t.Errorf("audio format = %d, want 1 (PCM)", got)
	}
	if got := binary.LittleEndian.Uint16(wav[22:24]); got != 1 {
		t.Errorf("channels = %d, want 1", got)
	}
	if got := binary.LittleEndian.Uint32(wav[24:28]); got != 24000 {
		t.Errorf("sample rate = %d, want 24000", got)
	}
	if got := binary.LittleEndian.Uint16(wav[34:36]); got != 16 {
		t.Errorf("bits per sample = %d, want 16", got)
	}
	// Byte rate = sampleRate * channels * bits/8 = 24000 * 1 * 2.
	if got := binary.LittleEndian.Uint32(wav[28:32]); got != 48000 {
		t.Errorf("byte rate = %d, want 48000", got)
	}
	// Data chunk size = dataLen.
	if got := binary.LittleEndian.Uint32(wav[40:44]); got != uint32(len(pcm)) {
		t.Errorf("data size = %d, want %d", got, len(pcm))
	}
	// Payload preserved.
	if !bytes.Equal(wav[44:], pcm) {
		t.Errorf("pcm payload not preserved")
	}
}
