// equate-core hosts the appliance's logical Poller and Ingestion components in
// one process. Events cross that boundary through a durable in-process queue;
// no broker is required for the single-node appliance.
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/equate/ogsd/services/ingestion-service/ingestion"
	"github.com/equate/ogsd/services/snmp-collector/internal/buffer"
	"github.com/equate/ogsd/services/snmp-collector/internal/config"
	"github.com/equate/ogsd/services/snmp-collector/internal/control"
	"github.com/equate/ogsd/services/snmp-collector/internal/heartbeat"
	"github.com/equate/ogsd/services/snmp-collector/internal/metrics"
	"github.com/equate/ogsd/services/snmp-collector/internal/poller"
	"github.com/equate/ogsd/services/snmp-collector/internal/publisher"
	"gopkg.in/yaml.v3"
)

var (
	buildVersion   = "unknown"
	buildGitCommit = "unknown"
	buildTime      = "unknown"
)

type applicationConfig struct {
	Database struct {
		URLEnv      string        `yaml:"url_env"`
		MaxConns    int32         `yaml:"max_conns"`
		MinConns    int32         `yaml:"min_conns"`
		MaxLifetime time.Duration `yaml:"max_lifetime"`
	} `yaml:"database"`
}

func main() {
	configPath := flag.String("config", "/etc/equate/application.yaml", "path to Equate application configuration")
	flag.Parse()

	log := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo}))
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Error("load collector configuration", "err", err)
		os.Exit(1)
	}
	if cfg.Publisher.Mode != "inprocess" {
		log.Error("invalid appliance transport", "mode", cfg.Publisher.Mode)
		os.Exit(1)
	}
	dbCfg, err := loadApplicationConfig(*configPath)
	if err != nil {
		log.Error("load application configuration", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	manager, err := config.NewManager(*configPath, cfg)
	if err != nil {
		log.Error("create configuration manager", "err", err)
		os.Exit(1)
	}
	m := metrics.New()
	spool, err := buffer.Open(buffer.Options{
		Path:          cfg.Buffer.Path,
		MaxEntries:    cfg.Buffer.MaxEntries,
		BusyTimeoutMS: cfg.Buffer.BusyTimeoutMS,
		Metrics:       m,
	})
	if err != nil {
		log.Error("open durable event spool", "err", err)
		os.Exit(1)
	}
	defer spool.Close() //nolint:errcheck

	ingestionRuntime, err := ingestion.Open(ctx, ingestion.Config{
		DatabaseURL: dbCfg.databaseURL(),
		MaxConns:    dbCfg.Database.MaxConns,
		MinConns:    dbCfg.Database.MinConns,
		MaxLifetime: dbCfg.Database.MaxLifetime,
	}, log)
	if err != nil {
		log.Error("open in-process ingestion", "err", err)
		os.Exit(1)
	}
	defer ingestionRuntime.Close()

	pub, err := publisher.NewInProcessPublisher(spool, ingestionRuntime, log, cfg.Buffer.BatchSize, cfg.Buffer.IdleBackoff)
	if err != nil {
		log.Error("create in-process dispatcher", "err", err)
		os.Exit(1)
	}
	p := poller.NewWithConfigSource(manager, pub, m, log)
	p.StatusStore().SetRevision(config.ConfigRevision(cfg))

	var controlServer *control.Server
	if cfg.Admin.ControlSocket != "" {
		stateDir := filepath.Dir(cfg.Buffer.Path)
		auditPath := filepath.Join(stateDir, "application.audit.log")
		if managed := cfg.ManagedInventoryPath(); managed != "" {
			stateDir = filepath.Dir(managed)
			auditPath = managed + ".audit.log"
		}
		controlServer, err = control.NewServer(control.Options{
			SocketPath: cfg.Admin.ControlSocket,
			Manager:    manager,
			Status:     p.StatusStore(),
			Health:     p.Tracker(),
			AuditPath:  auditPath,
			StateDir:   stateDir,
			Log:        log,
		})
		if err != nil {
			log.Error("create collector control server", "err", err)
			os.Exit(1)
		}
		if err := controlServer.Listen(); err != nil {
			log.Error("listen collector control socket", "err", err)
			os.Exit(1)
		}
		go func() {
			if err := controlServer.Serve(ctx); err != nil && ctx.Err() == nil {
				log.Error("collector control server failed", "err", err)
				stop()
			}
		}()
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		if spool.Available() {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok\n"))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("not ready\n"))
	})
	srv := &http.Server{Addr: cfg.Admin.Listen, Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("core admin server failed", "err", err)
			stop()
		}
	}()

	go pub.Run(ctx)
	go p.Run(ctx)
	hb := heartbeat.New(manager, pub, m, log, func() (int64, error) { return pub.Depth(), nil }, heartbeat.BuildInfo{
		Version: buildVersion, GitCommit: buildGitCommit, BuildTime: buildTime,
	})
	go hb.Run(ctx)
	go reloadOnHUP(ctx, manager, p, log)

	log.Info("equate core started", "transport", "inprocess", "spool", cfg.Buffer.Path)
	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if controlServer != nil {
		_ = controlServer.Close()
	}
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Warn("shutdown core admin server", "err", err)
	}
}

func loadApplicationConfig(path string) (applicationConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return applicationConfig{}, fmt.Errorf("read application config: %w", err)
	}
	var cfg applicationConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return applicationConfig{}, fmt.Errorf("parse application config: %w", err)
	}
	if cfg.Database.URLEnv == "" {
		cfg.Database.URLEnv = "DATABASE_URL"
	}
	if cfg.Database.MaxConns <= 0 {
		cfg.Database.MaxConns = 10
	}
	if cfg.Database.MaxLifetime <= 0 {
		cfg.Database.MaxLifetime = time.Hour
	}
	if cfg.databaseURL() == "" {
		return applicationConfig{}, fmt.Errorf("environment variable %q is required", cfg.Database.URLEnv)
	}
	return cfg, nil
}

func (c applicationConfig) databaseURL() string {
	return os.Getenv(c.Database.URLEnv)
}

func reloadOnHUP(ctx context.Context, manager *config.Manager, p *poller.Poller, log *slog.Logger) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGHUP)
	defer signal.Stop(ch)
	for {
		select {
		case <-ctx.Done():
			return
		case <-ch:
			if err := manager.Reload(); err != nil {
				log.Error("application configuration reload failed", "err", err)
				continue
			}
			p.StatusStore().SetRevision(config.ConfigRevision(manager.Current()))
			log.Info("application configuration reloaded")
		}
	}
}
