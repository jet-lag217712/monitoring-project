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

	"github.com/equate/ogsd/services/snmp-collector/internal/buffer"
	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/equate/ogsd/services/snmp-collector/internal/poller"
	"github.com/equate/ogsd/services/snmp-collector/internal/publisher"
)

func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "validate":
			os.Exit(runValidate(os.Args[2:]))
		case "discover":
			os.Exit(runDiscover(os.Args[2:]))
		}
	}

	configPath := flag.String("config", "configs/collector.example.yaml", "path to collector config file")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("load config", "err", err)
		os.Exit(1)
	}
	configManager, err := config.NewManager(*configPath, cfg)
	if err != nil {
		log.Error("create config manager", "err", err)
		os.Exit(1)
	}

	m := metrics.New()

	// mqttCtx outlives the poller so we can drain the buffer after SIGTERM.
	mqttCtx, mqttCancel := context.WithCancel(context.Background())
	defer mqttCancel()

	pollCtx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	reloadCh := make(chan os.Signal, 1)
	signal.Notify(reloadCh, syscall.SIGHUP)
	defer signal.Stop(reloadCh)

	pub, shutdownPub, err := buildPublisher(mqttCtx, cfg, m, log)
	if err != nil {
		log.Error("build publisher", "err", err)
		os.Exit(1)
	}

	p := poller.NewWithConfigSource(configManager, pub, m, log)

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
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

	log.Info("collector starting",
		"site_id", cfg.SiteID,
		"devices", len(cfg.Devices),
		"poll_interval", cfg.PollInterval.String(),
		"max_workers", cfg.MaxWorkers,
		"publisher_mode", cfg.Publisher.Mode,
	)

	go p.Run(pollCtx)
	go func() {
		for {
			select {
			case <-pollCtx.Done():
				return
			case <-reloadCh:
				if err := configManager.Reload(); err != nil {
					m.ConfigReloadFailureTotal.Inc()
					log.Error("configuration reload failed", "err", err)
					continue
				}
				m.ConfigReloadSuccessTotal.Inc()
				active := configManager.Current()
				log.Info("configuration reloaded", "devices", len(active.Devices), "poll_interval", active.PollInterval.String())
			}
		}
	}()

	<-pollCtx.Done()
	log.Info("shutting down")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error("admin server shutdown", "err", err)
	}

	if err := shutdownPub(shutdownCtx); err != nil {
		log.Warn("publisher shutdown", "err", err)
	}
	mqttCancel()
}

func runValidate(args []string) int {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/collector.example.yaml", "path to collector config file")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if fs.NArg() != 0 {
		fmt.Fprintln(os.Stderr, "validate accepts no positional arguments")
		return 2
	}
	if _, err := config.LoadForValidation(*configPath); err != nil {
		fmt.Fprintf(os.Stderr, "configuration invalid: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "configuration valid: %s\n", *configPath)
	return 0
}

func buildPublisher(mqttCtx context.Context, cfg *config.Config, m *metrics.Collector, log *slog.Logger) (publisher.Publisher, func(context.Context) error, error) {
	switch cfg.Publisher.Mode {
	case "stdout":
		return publisher.NewStdoutPublisher(), func(context.Context) error { return nil }, nil
	case "mqtt":
		store, err := buffer.Open(buffer.Options{
			Path:          cfg.Buffer.Path,
			MaxEntries:    cfg.Buffer.MaxEntries,
			BusyTimeoutMS: cfg.Buffer.BusyTimeoutMS,
			Metrics:       m,
		})
		if err != nil {
			return nil, nil, fmt.Errorf("open buffer: %w", err)
		}

		mqttClient, err := publisher.NewMQTTClient(mqttCtx, cfg.MQTT, cfg.MQTTPassword(), m, log)
		if err != nil {
			_ = store.Close()
			return nil, nil, fmt.Errorf("mqtt client: %w", err)
		}

		flusherCtx, stopFlusher := context.WithCancel(mqttCtx)
		bp := publisher.NewBufferedPublisher(store, mqttClient, m, log, cfg.Buffer.BatchSize, cfg.Buffer.IdleBackoff)
		go bp.RunFlusher(flusherCtx)

		shutdown := func(ctx context.Context) error {
			stopFlusher()
			drainErr := bp.Drain(ctx)
			if err := store.Close(); err != nil {
				if drainErr != nil {
					return fmt.Errorf("drain: %v; close buffer: %w", drainErr, err)
				}
				return fmt.Errorf("close buffer: %w", err)
			}
			return drainErr
		}
		return bp, shutdown, nil
	default:
		return nil, nil, fmt.Errorf("unknown publisher mode %q", cfg.Publisher.Mode)
	}
}
