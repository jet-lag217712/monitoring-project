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

	"github.com/equate/ogsd/services/ingestion-service/internal/config"
	"github.com/equate/ogsd/services/ingestion-service/internal/handler"
	"github.com/equate/ogsd/services/ingestion-service/internal/metrics"
	"github.com/equate/ogsd/services/ingestion-service/internal/mqtt"
	"github.com/equate/ogsd/services/ingestion-service/internal/store"
)

func main() {
	configPath := flag.String("config", "configs/ingestion.example.yaml", "path to ingestion config file")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("load config", "err", err)
		os.Exit(1)
	}

	m := metrics.New()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	db, err := store.Open(ctx, cfg.DatabaseURL(), cfg.Database.MaxConns, cfg.Database.MinConns, cfg.Database.MaxLifetime)
	if err != nil {
		log.Error("open database", "err", err)
		os.Exit(1)
	}
	defer db.Close()

	h := handler.New(db, m, log)

	sub, err := mqtt.NewSubscriber(ctx, cfg.MQTT, cfg.MQTTPassword(), h, m, log)
	if err != nil {
		log.Error("mqtt subscriber", "err", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if !sub.IsReady() {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("not ready\n"))
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	srv := &http.Server{
		Addr:              cfg.Admin.Listen,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Info("admin server listening", "addr", cfg.Admin.Listen)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("admin server failed", "err", err)
			stop()
		}
	}()

	go func() {
		if err := sub.AwaitReady(ctx); err != nil {
			log.Error("await mqtt ready", "err", err)
			stop()
			return
		}
		log.Info("ingestion ready", "broker", cfg.MQTT.Broker, "topic", cfg.MQTT.Topic)
	}()

	<-ctx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("admin server shutdown", "err", err)
	}
	<-sub.Done()
}
