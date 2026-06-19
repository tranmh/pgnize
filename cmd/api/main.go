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
	"github.com/tranmh/pgnize/internal/config"
	"github.com/tranmh/pgnize/internal/httpapi"
	"github.com/tranmh/pgnize/internal/jobs"
	"github.com/tranmh/pgnize/internal/recognition"
	"github.com/tranmh/pgnize/internal/storage"
	"github.com/tranmh/pgnize/internal/store"
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

	// Migrations run on every boot (mirrors swiss-manager migrate-on-start).
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
	rec := buildRecognizer(cfg)
	slog.Info("recognizer", "name", rec.Name())

	srv := &httpapi.Server{Cfg: cfg, Store: st, Storage: blob, Recognizer: rec}

	// In-process recognition worker pool.
	pool := &jobs.Pool{
		Deps: jobs.Deps{Store: st, Storage: blob, Recognizer: rec, FewShotMax: cfg.FewShotMax},
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

func buildRecognizer(cfg config.Config) recognition.Recognizer {
	switch cfg.Recognizer {
	case "ollama":
		return recognition.NewOllama(cfg.OllamaHost, cfg.RecognizerModel)
	default:
		return recognition.NewFake()
	}
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
