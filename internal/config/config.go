// Package config loads pgnize configuration from the environment.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is the fully-resolved runtime configuration.
type Config struct {
	DatabaseURL string
	AuthSecret  string
	APIAddr     string
	PublicBase  string

	StorageDriver string // auto | s3 | filesystem
	StorageDir    string
	S3Endpoint    string
	S3Region      string
	S3AccessKey   string
	S3SecretKey   string
	S3Bucket      string
	S3PathStyle   bool

	Recognizer        string // fake | ollama | gemini
	RecognizerModel   string
	OllamaHost        string
	GeminiAPIKey      string // Google AI Studio API key; enables the gemini backend
	GeminiModel       string
	GeminiHost        string
	RecognitionWorker int
	FewShotMax        int

	// Text-to-speech (coach voice). Gemini is primary when GeminiAPIKey is set; Piper is the
	// self-hosted fallback (chained in when PiperHost is set, or sole backend otherwise).
	PiperHost      string
	GeminiTTSModel string
	TTSGeminiVoice string
	PiperVoice     string

	// Server-side chess engine for the conversational coach. ENGINE=stockfish opts into the
	// UCI binary; otherwise the deterministic fake engine is used (tests/CI/dev without a
	// binary). The remaining knobs tune the Stockfish pool.
	Engine              string // fake | stockfish
	EnginePath          string
	EngineInstances     int
	EngineThreads       int
	EngineHashMB        int
	EngineMoveTimeMs    int
	EngineMaxMoveTimeMs int

	// Conversational coach (function-calling chat). The backend mirrors the recognizer/coach
	// selection (Gemini when GeminiAPIKey is set, else Ollama when RECOGNIZER=ollama, else fake).
	ChatMaxToolIters int

	// Speech-to-text for voice questions. STT=gemini uses Gemini multimodal audio (needs
	// GeminiAPIKey); otherwise the deterministic fake transcript is used (tests/CI).
	STT         string
	STTModel    string
	STTMaxBytes int64

	UploadMaxBytes int64
	AnonUploadTTLd int

	// RateLimitDisabled turns off all per-IP/per-user rate limiting. Intended for
	// local automated testing only — defaults to false and must never be set in prod.
	RateLimitDisabled bool
}

// Load reads configuration from environment variables, applying defaults.
func Load() (Config, error) {
	c := Config{
		DatabaseURL:       env("DATABASE_URL", "postgres://pgnize:pgnize@localhost:5432/pgnize?sslmode=disable"),
		AuthSecret:        env("AUTH_SECRET", ""),
		APIAddr:           env("API_ADDR", ":8080"),
		PublicBase:        env("PUBLIC_BASE_URL", "http://localhost:8080"),
		StorageDriver:     env("STORAGE_DRIVER", "auto"),
		StorageDir:        env("STORAGE_DIR", "./data/uploads"),
		S3Endpoint:        env("S3_ENDPOINT", ""),
		S3Region:          env("S3_REGION", "us-east-1"),
		S3AccessKey:       env("S3_ACCESS_KEY", ""),
		S3SecretKey:       env("S3_SECRET_KEY", ""),
		S3Bucket:          env("S3_BUCKET", ""),
		S3PathStyle:       envBool("S3_FORCE_PATH_STYLE", true),
		Recognizer:        env("RECOGNIZER", "fake"),
		RecognizerModel:   env("RECOGNIZER_MODEL", "minicpm-v"),
		OllamaHost:        env("OLLAMA_HOST", "http://localhost:11434"),
		GeminiAPIKey:      env("GEMINI_API_KEY", ""),
		GeminiModel:       env("GEMINI_MODEL", "gemini-2.5-flash"),
		GeminiHost:        env("GEMINI_HOST", "https://generativelanguage.googleapis.com"),
		RecognitionWorker: envInt("RECOGNITION_WORKERS", 2),
		FewShotMax:        envInt("FEWSHOT_MAX_EXAMPLES", 3),
		PiperHost:         env("PIPER_HOST", ""),
		GeminiTTSModel:    env("GEMINI_TTS_MODEL", "gemini-2.5-flash-preview-tts"),
		TTSGeminiVoice:    env("TTS_GEMINI_VOICE", "Kore"),
		PiperVoice:        env("PIPER_VOICE", "de_DE-thorsten-medium"),

		Engine:              env("ENGINE", "fake"),
		EnginePath:          env("ENGINE_PATH", "stockfish"),
		EngineInstances:     envInt("ENGINE_INSTANCES", 2),
		EngineThreads:       envInt("ENGINE_THREADS", 1),
		EngineHashMB:        envInt("ENGINE_HASH_MB", 64),
		EngineMoveTimeMs:    envInt("ENGINE_MOVETIME_MS", 300),
		EngineMaxMoveTimeMs: envInt("ENGINE_MAX_MOVETIME_MS", 2000),
		ChatMaxToolIters:    envInt("CHAT_MAX_TOOL_ITERS", 5),
		STT:                 env("STT", "fake"),
		STTModel:            env("STT_MODEL", "gemini-2.5-flash"),
		STTMaxBytes:         int64(envInt("STT_MAX_BYTES", 5<<20)),

		UploadMaxBytes:    int64(envInt("UPLOAD_MAX_BYTES", 15<<20)),
		AnonUploadTTLd:    envInt("ANON_UPLOAD_TTL_DAYS", 7),
		RateLimitDisabled: envBool("RATE_LIMIT_DISABLED", false),
	}
	if c.AuthSecret == "" {
		return c, fmt.Errorf("AUTH_SECRET is required")
	}
	if len(c.AuthSecret) < 16 {
		return c, fmt.Errorf("AUTH_SECRET must be at least 16 bytes")
	}
	return c, nil
}

// ResolveStorageDriver collapses "auto" into a concrete driver.
func (c Config) ResolveStorageDriver() string {
	if c.StorageDriver == "auto" {
		if c.S3Bucket != "" {
			return "s3"
		}
		return "filesystem"
	}
	return c.StorageDriver
}

func env(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
			return n
		}
	}
	return def
}

func envBool(k string, def bool) bool {
	if v, ok := os.LookupEnv(k); ok {
		b, err := strconv.ParseBool(strings.TrimSpace(v))
		if err == nil {
			return b
		}
	}
	return def
}
