package main

import (
	"context"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/config"
	"github.com/equate/ogsd/services/backend-api/internal/handlers"
	"github.com/equate/ogsd/services/backend-api/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	configPath := flag.String("config", "configs/api.example.yaml", "path to API config file")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL(), cfg.Database.MaxConns, cfg.Database.MinConns, cfg.Database.MaxLifetime)
	if err != nil {
		log.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	api := handlers.New(db, log, cfg.OnlineThreshold)
	mux := http.NewServeMux()
	api.Register(mux)

	var apiHandler http.Handler = mux
	apiHandler = handlers.NormalizePath(apiHandler)
	apiHandler = handlers.RequestLog(log, apiHandler)
	if origins := cfg.CORSOriginList(); len(origins) > 0 {
		apiHandler = handlers.CORS(origins, apiHandler)
	}

	apiSrv := &http.Server{
		Addr:              cfg.API.Listen,
		Handler:           apiHandler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	adminMux := http.NewServeMux()
	adminMux.Handle("/metrics", promhttp.Handler())
	adminMux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	adminSrv := &http.Server{
		Addr:              cfg.Admin.Listen,
		Handler:           adminMux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("api server listening", "addr", cfg.API.Listen)
		if err := apiSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("api server failed", "err", err)
			stop()
		}
	}()

	go func() {
		log.Info("admin server listening", "addr", cfg.Admin.Listen)
		if err := adminSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("admin server failed", "err", err)
			stop()
		}
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := apiSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("api server shutdown", "err", err)
	}
	if err := adminSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("admin server shutdown", "err", err)
	}
}
