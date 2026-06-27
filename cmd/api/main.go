// Command api is the pgnize backend: REST server + in-process recognition worker pool.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/tranmh/pgnize/internal/auth"
	"github.com/tranmh/pgnize/internal/chat"
	"github.com/tranmh/pgnize/internal/coaching"
	"github.com/tranmh/pgnize/internal/config"
	"github.com/tranmh/pgnize/internal/engine"
	"github.com/tranmh/pgnize/internal/httpapi"
	"github.com/tranmh/pgnize/internal/jobs"
	"github.com/tranmh/pgnize/internal/recognition"
	"github.com/tranmh/pgnize/internal/storage"
	"github.com/tranmh/pgnize/internal/store"
	"github.com/tranmh/pgnize/internal/stt"
	"github.com/tranmh/pgnize/internal/tts"
	"github.com/tranmh/pgnize/migrations"
)

func main() {
	var (
		migrateOnly = flag.Bool("migrate-only", false, "run migrations then exit")
		migrateDown = flag.Bool("migrate-down", false, "roll back one migration then exit")
		seed        = flag.Bool("seed", false, "seed a demo user then exit")
		healthcheck = flag.Bool("healthcheck", false, "probe /healthz then exit (docker HEALTHCHECK)")
	)
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		if *healthcheck {
			os.Exit(probeHealth(":8080"))
		}
		fatal("config", err)
	}
	if *healthcheck {
		os.Exit(probeHealth(cfg.APIAddr))
	}

	// Migrations run on every boot (mirrors OpenPairing.org migrate-on-start).
	if err := store.Migrate(cfg.DatabaseURL, migrations.FS, *migrateDown); err != nil {
		fatal("migrate", err)
	}
	if *migrateOnly || *migrateDown {
		slog.Info("migrations applied")
		return
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	st, err := store.New(ctx, cfg.DatabaseURL)
	if err != nil {
		fatal("store", err)
	}
	defer st.Close()

	if *seed {
		if err := seedDemo(ctx, st); err != nil {
			fatal("seed", err)
		}
		slog.Info("seeded demo user demo@pgnize.local / demo1234")
		return
	}

	blob, err := storage.New(ctx, cfg)
	if err != nil {
		fatal("storage", err)
	}
	reg := buildRegistry(cfg)
	slog.Info("recognizers", "default", reg.Default(), "available", reg.Advertised())

	coach := buildCoach(cfg)
	slog.Info("coach", "backend", coach.Name())

	speaker := buildTTS(cfg)
	slog.Info("tts", "backend", speaker.Name())

	eng := buildEngine(cfg)
	defer eng.Close()
	slog.Info("engine", "backend", eng.Name())

	chatter := buildChat(cfg, eng)
	slog.Info("chat", "backend", chatter.Name())

	transcriber := buildSTT(cfg)
	slog.Info("stt", "backend", sttName(transcriber))

	srv := &httpapi.Server{
		Cfg: cfg, Store: st, Storage: blob, Recognizers: reg,
		Coach: coach, TTS: speaker, Chat: chatter, STT: transcriber,
	}

	// In-process recognition worker pool.
	pool := &jobs.Pool{
		Deps:    jobs.Deps{Store: st, Storage: blob, Registry: reg, FewShotMax: cfg.FewShotMax},
		Workers: cfg.RecognitionWorker,
	}
	go pool.Run(ctx)

	httpSrv := &http.Server{Addr: cfg.APIAddr, Handler: srv.Routes(), ReadHeaderTimeout: 10 * time.Second}
	go func() {
		slog.Info("api listening", "addr", cfg.APIAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fatal("listen", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(shutCtx)
}

// buildRegistry assembles the selectable recognition backends. The backend named by
// RECOGNIZER is always registered; Gemini is added when GEMINI_API_KEY is set and, when
// present, becomes the default. The deterministic fake backend is only advertised when no
// real backend is configured (so it stays available for tests/CI without leaking into prod).
func buildRegistry(cfg config.Config) *recognition.Registry {
	reg := recognition.NewRegistry()
	hasGemini := cfg.GeminiAPIKey != ""

	switch cfg.Recognizer {
	case "ollama":
		reg.Register("ollama", "Ollama (local model)", true,
			recognition.NewOllama(cfg.OllamaHost, cfg.RecognizerModel))
	default: // "fake", "gemini", or unset → deterministic fallback recognizer
		reg.Register("fake", "Built-in test recognizer", !hasGemini, recognition.NewFake())
	}

	if hasGemini {
		reg.Register("gemini", "Gemini Flash (Google)", true,
			recognition.NewGemini(cfg.GeminiHost, cfg.GeminiModel, cfg.GeminiAPIKey))
	}

	// Default: Gemini when configured, else the configured RECOGNIZER (fake fallback).
	switch {
	case hasGemini:
		reg.SetDefault("gemini")
	case cfg.Recognizer == "ollama":
		reg.SetDefault("ollama")
	default:
		reg.SetDefault("fake")
	}
	return reg
}

// buildCoach selects the engine→prose coach backend from the SAME configuration as the
// recognizer (buildRegistry) — there is no separate COACH knob. Gemini when GEMINI_API_KEY
// is set (it is then also the default recognizer), Ollama when RECOGNIZER=ollama, else the
// deterministic fake (tests/CI).
func buildCoach(cfg config.Config) coaching.Coach {
	switch {
	case cfg.GeminiAPIKey != "":
		return coaching.NewGeminiCoach(cfg.GeminiHost, cfg.GeminiModel, cfg.GeminiAPIKey)
	case cfg.Recognizer == "ollama":
		return coaching.NewOllamaCoach(cfg.OllamaHost, cfg.RecognizerModel)
	default:
		return coaching.NewFake()
	}
}

// buildTTS selects the coach-voice synthesizer, mirroring buildCoach. Gemini is primary
// when GEMINI_API_KEY is set; a configured PIPER_HOST is chained in as the local fallback
// (or used alone when no Gemini key is present). With neither configured, the deterministic
// fake is used (tests/CI), and the handler returns 503 so the client falls back to browser
// speech.
func buildTTS(cfg config.Config) tts.Synthesizer {
	var synths []tts.Synthesizer
	if cfg.GeminiAPIKey != "" {
		synths = append(synths, tts.NewGeminiTTS(cfg.GeminiHost, cfg.GeminiTTSModel, cfg.GeminiAPIKey, cfg.TTSGeminiVoice))
	}
	if cfg.PiperHost != "" {
		synths = append(synths, tts.NewPiperTTS(cfg.PiperHost, cfg.PiperVoice))
	}
	switch len(synths) {
	case 0:
		return tts.NewFake()
	case 1:
		return synths[0]
	default:
		return tts.NewChain(synths...)
	}
}

// buildEngine selects the server-side chess engine for the conversational coach. ENGINE=
// stockfish starts a UCI subprocess pool; if the binary is missing/unresponsive it logs and
// falls back to the deterministic fake so the API still boots (the coach just can't analyze).
func buildEngine(cfg config.Config) engine.Engine {
	if cfg.Engine == "stockfish" {
		eng, err := engine.NewStockfish(cfg.EnginePath, engine.PoolOpts{
			Instances:     cfg.EngineInstances,
			Threads:       cfg.EngineThreads,
			HashMB:        cfg.EngineHashMB,
			MoveTimeMs:    cfg.EngineMoveTimeMs,
			MaxMoveTimeMs: cfg.EngineMaxMoveTimeMs,
		})
		if err != nil {
			slog.Warn("stockfish unavailable, using fake engine", "err", err)
			return engine.NewFake()
		}
		return eng
	}
	return engine.NewFake()
}

// buildChat selects the conversational-coach backend, mirroring buildCoach: Gemini function-
// calling when GEMINI_API_KEY is set, Ollama tool-calling when RECOGNIZER=ollama, else the
// deterministic fake (which still drives the real engine tool loop for tests/CI).
func buildChat(cfg config.Config, eng engine.Engine) chat.Chatter {
	switch {
	case cfg.GeminiAPIKey != "":
		return chat.NewGemini(cfg.GeminiHost, cfg.GeminiModel, cfg.GeminiAPIKey, eng)
	case cfg.Recognizer == "ollama":
		return chat.NewOllama(cfg.OllamaHost, cfg.RecognizerModel, eng)
	default:
		return chat.NewFake(eng)
	}
}

// buildSTT selects the speech-to-text backend. STT=gemini uses Gemini multimodal audio (needs
// a key); otherwise the deterministic fake transcript is used (tests/CI).
func buildSTT(cfg config.Config) stt.Transcriber {
	if cfg.STT == "gemini" && cfg.GeminiAPIKey != "" {
		return stt.NewGemini(cfg.GeminiHost, cfg.STTModel, cfg.GeminiAPIKey)
	}
	return stt.NewFake()
}

func sttName(t stt.Transcriber) string {
	if t == nil {
		return "disabled"
	}
	return t.Name()
}

func seedDemo(ctx context.Context, st *store.Store) error {
	hash, err := auth.HashPassword("demo1234")
	if err != nil {
		return err
	}
	_, err = st.CreateUser(ctx, "Demo User", "demo@pgnize.local", hash)
	if err == store.ErrEmailTaken {
		return nil
	}
	return err
}

func probeHealth(addr string) int {
	url := "http://localhost" + addr + "/healthz"
	resp, err := http.Get(url) //nolint:gosec // local liveness probe
	if err != nil || resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}

func fatal(stage string, err error) {
	fmt.Fprintf(os.Stderr, "fatal: %s: %v\n", stage, err)
	os.Exit(1)
}
