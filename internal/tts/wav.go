package tts

import (
	"bytes"
	"encoding/binary"
)

// pcmToWAV wraps raw signed 16-bit little-endian mono PCM samples in a canonical 44-byte
// WAV (RIFF) header. Gemini returns headerless PCM; browsers and <audio> need a container.
func pcmToWAV(pcm []byte, sampleRate, bitsPerSample, channels int) []byte {
	byteRate := sampleRate * channels * bitsPerSample / 8
	blockAlign := channels * bitsPerSample / 8
	dataLen := len(pcm)

	var buf bytes.Buffer
	// RIFF chunk descriptor.
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(36+dataLen)) // file size - 8
	buf.WriteString("WAVE")
	// fmt sub-chunk.
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, uint32(16)) // PCM fmt chunk size
	binary.Write(&buf, binary.LittleEndian, uint16(1))  // audio format: PCM
	binary.Write(&buf, binary.LittleEndian, uint16(channels))
	binary.Write(&buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(&buf, binary.LittleEndian, uint32(byteRate))
	binary.Write(&buf, binary.LittleEndian, uint16(blockAlign))
	binary.Write(&buf, binary.LittleEndian, uint16(bitsPerSample))
	// data sub-chunk.
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, uint32(dataLen))
	buf.Write(pcm)
	return buf.Bytes()
}
