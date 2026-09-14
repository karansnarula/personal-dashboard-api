package api

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/karansnarula/personal-dashboard-api/internal/api/handler"
	"github.com/karansnarula/personal-dashboard-api/internal/api/middleware"
)

// Config is the subset of app configuration the HTTP layer needs.
type Config struct {
	Port               int
	Development        bool
	CORSAllowedOrigins []string
}

// Deps are the collaborators handlers need. main wires them; nothing in
// this package constructs a database or service itself.
type Deps struct {
	Logger *slog.Logger
	DB     handler.Pinger
	Tokens middleware.TokenParser

	AuthService      handler.AuthService
	WidgetService    handler.WidgetService
	DashboardService handler.DashboardService
}

// NewServer returns a configured *http.Server. Timeouts are set so a slow or
// malicious client cannot hold connections open indefinitely.
func NewServer(cfg Config, deps Deps) *http.Server {
	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           NewRouter(cfg, deps),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second, // dashboard fan-out can be slow while it is sequential
		IdleTimeout:       120 * time.Second,
	}
}
