// Command api is the entrypoint for the personal dashboard API. It only
// wires things together: config → logger → database → server → run.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/karansnarula/personal-dashboard-api/internal/api"
	"github.com/karansnarula/personal-dashboard-api/internal/auth"
	"github.com/karansnarula/personal-dashboard-api/internal/config"
	"github.com/karansnarula/personal-dashboard-api/internal/database"
	"github.com/karansnarula/personal-dashboard-api/internal/repository/postgres"
	"github.com/karansnarula/personal-dashboard-api/internal/service"
	"github.com/karansnarula/personal-dashboard-api/internal/widget"
)

const shutdownTimeout = 10 * time.Second

func main() {
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	logger := newLogger(cfg)
	slog.SetDefault(logger)

	// ctx is cancelled on SIGINT/SIGTERM; everything long-lived hangs off it.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	logger.Info("database connected")

	if err := database.Migrate(ctx, pool, logger); err != nil {
		return err
	}

	warnMissingKeys(cfg, logger)

	userRepo := postgres.NewUserRepository(pool)
	widgetRepo := postgres.NewWidgetRepository(pool)
	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	clients := widget.NewRegistry(widget.Config{
		OpenWeatherMapKey: cfg.OpenWeatherMapKey,
		NewsAPIKey:        cfg.NewsAPIKey,
		FinnhubKey:        cfg.FinnhubKey,
	})

	srv := api.NewServer(
		api.Config{
			Port:               cfg.HTTPPort,
			Development:        cfg.IsDevelopment(),
			CORSAllowedOrigins: cfg.CORSAllowedOrigins,
		},
		api.Deps{
			Logger:        logger,
			DB:            pool,
			Tokens:        tokens,
			AuthService:   service.NewAuthService(userRepo, tokens),
			WidgetService: service.NewWidgetService(widgetRepo),
			DashboardService: service.NewDashboardService(
				widgetRepo, clients, cfg.ExternalTimeout, logger,
			),
		},
	)

	errCh := make(chan error, 1)
	go func() {
		logger.Info("server listening", "addr", srv.Addr, "env", cfg.AppEnv)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("server: %w", err)
	case <-ctx.Done():
	}

	logger.Info("shutting down", "timeout", shutdownTimeout)
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	logger.Info("server stopped")
	return nil
}

func newLogger(cfg config.Config) *slog.Logger {
	level := slog.LevelInfo
	if cfg.IsDevelopment() {
		level = slog.LevelDebug
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func warnMissingKeys(cfg config.Config, logger *slog.Logger) {
	keys := map[string]string{
		"OPENWEATHERMAP_API_KEY": cfg.OpenWeatherMapKey,
		"NEWSAPI_API_KEY":        cfg.NewsAPIKey,
		"FINNHUB_API_KEY":        cfg.FinnhubKey,
	}
	for name, value := range keys {
		if value == "" {
			logger.Warn("external API key not set; that widget type will report an error", "key", name)
		}
	}
}
