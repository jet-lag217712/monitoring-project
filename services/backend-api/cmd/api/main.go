package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/equate/ogsd/services/backend-api/internal/auth"
	"github.com/equate/ogsd/services/backend-api/internal/config"
	"github.com/equate/ogsd/services/backend-api/internal/handlers"
	"github.com/equate/ogsd/services/backend-api/internal/store"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "admin" {
		os.Exit(runAdmin(os.Args[2:]))
	}
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
	if cfg.AuthEnabled() {
		switch cfg.Auth.Mode {
		case "local":
			sessions := auth.NewSessionAuthenticator(db.Pool(), auth.SessionOptions{
				CookieName: cfg.Auth.CookieName,
				TTL:        cfg.Auth.SessionTTL,
				Secure:     cfg.Auth.CookieSecure,
			}, log)
			sessions.Register(mux)
			apiHandler = sessions.RequireSession(apiHandler)
			log.Info("local session auth enabled")
		case "google_bearer":
			verifier, err := auth.NewGoogleVerifier(ctx, cfg.GoogleClientID())
			if err != nil {
				log.Error("init google oidc verifier", "err", err)
				os.Exit(1)
			}
			apiHandler = auth.RequireGoogleOIDC(verifier, log, apiHandler)
			log.Info("google oidc auth enabled")
		case "google_session":
			if len(cfg.Auth.AllowedDomains) == 0 {
				bootstrap := auth.NewBootstrapAuthenticator()
				bootstrap.Register(mux)
				apiHandler = bootstrap.RequireSetup(apiHandler)
				log.Warn("google session setup required; dashboard access is locked")
				break
			}
			verifier, err := auth.NewGoogleVerifier(ctx, cfg.GoogleClientID())
			if err != nil {
				log.Error("init google oidc verifier", "err", err)
				os.Exit(1)
			}
			sessions, err := auth.NewGoogleSessionAuthenticator(db.Pool(), verifier, auth.GoogleSessionOptions{
				SessionOptions: auth.SessionOptions{
					CookieName: cfg.Auth.CookieName,
					TTL:        cfg.Auth.SessionTTL,
					Secure:     cfg.Auth.CookieSecure,
				},
				ClientID:       cfg.GoogleClientID(),
				AllowedDomains: cfg.Auth.AllowedDomains,
			}, log)
			if err != nil {
				log.Error("init google session auth", "err", err)
				os.Exit(1)
			}
			sessions.Register(mux)
			apiHandler = sessions.RequireSession(apiHandler)
			log.Info("google browser session auth enabled")
		case "oidc", "google":
			sessions, err := auth.NewOIDCSessionAuthenticator(ctx, db.Pool(), auth.OIDCOptions{
				Provider:      cfg.Auth.Mode,
				Issuer:        cfg.Auth.OIDCIssuer,
				ClientID:      cfg.Auth.OIDCClientID,
				ClientSecret:  os.Getenv(cfg.Auth.OIDCClientSecretEnv),
				RedirectURL:   cfg.Auth.OIDCRedirectURL,
				AllowedEmails: cfg.Auth.AllowedEmails,
				AllowedGroups: cfg.Auth.AllowedGroups,
				CookieName:    cfg.Auth.CookieName,
				SessionTTL:    cfg.Auth.SessionTTL,
				CookieSecure:  cfg.Auth.CookieSecure,
			}, log)
			if err != nil {
				log.Error("init browser oidc", "err", err)
				os.Exit(1)
			}
			sessions.Register(mux)
			apiHandler = sessions.RequireSession(apiHandler)
			log.Info("browser oidc auth enabled", "provider", cfg.Auth.Mode)
		}
	} else {
		log.Warn("google oidc auth disabled; /api/* is unauthenticated")
	}
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

func runAdmin(args []string) int {
	fs := flag.NewFlagSet("api admin", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	configPath := fs.String("config", "configs/api.example.yaml", "path to API config file")
	username := fs.String("username", "", "local administrator username")
	email := fs.String("email", "", "local administrator email")
	passwordStdin := fs.Bool("password-stdin", false, "read password from standard input")
	if err := fs.Parse(args); err != nil || fs.NArg() != 1 || fs.Arg(0) != "create-local-user" || !*passwordStdin {
		fmt.Fprintln(os.Stderr, "usage: api admin -config <path> -username <name> [-email <email>] -password-stdin create-local-user")
		return 2
	}
	password, err := bufio.NewReader(os.Stdin).ReadString('\n')
	if err != nil && len(password) == 0 {
		fmt.Fprintln(os.Stderr, "read password:", err)
		return 1
	}
	cfg, err := config.Load(*configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "load config:", err)
		return 1
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	db, err := store.Open(ctx, cfg.DatabaseURL(), cfg.Database.MaxConns, cfg.Database.MinConns, cfg.Database.MaxLifetime)
	if err != nil {
		fmt.Fprintln(os.Stderr, "open database:", err)
		return 1
	}
	defer db.Close()
	user, err := auth.CreateLocalUser(ctx, db.Pool(), *username, *email, strings.TrimRight(password, "\r\n"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "create local user:", err)
		return 1
	}
	fmt.Printf("created local administrator %s (%s)\n", user.Username, user.ID)
	return 0
}
