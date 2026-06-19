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

	UploadMaxBytes  int64
	AnonUploadTTLd  int
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
		UploadMaxBytes:    int64(envInt("UPLOAD_MAX_BYTES", 15<<20)),
		AnonUploadTTLd:    envInt("ANON_UPLOAD_TTL_DAYS", 7),
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
