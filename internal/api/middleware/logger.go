package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/reqctx"
)

// Logger emits one structured access-log line per request.
func Logger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		if q := c.Request.URL.RawQuery; q != "" {
			path += "?" + q
		}

		c.Next()

		status := c.Writer.Status()
		attrs := []slog.Attr{
			slog.String("request_id", reqctx.RequestID(c)),
			slog.String("method", c.Request.Method),
			slog.String("path", path),
			slog.Int("status", status),
			slog.Int64("latency_ms", time.Since(start).Milliseconds()),
			slog.Int("bytes", c.Writer.Size()),
			slog.String("ip", c.ClientIP()),
		}
		if uid := reqctx.UserID(c); uid != 0 {
			attrs = append(attrs, slog.Int64("user_id", uid))
		}
		if len(c.Errors) > 0 {
			attrs = append(attrs, slog.String("errors", c.Errors.String()))
		}

		level := slog.LevelInfo
		switch {
		case status >= http.StatusInternalServerError:
			level = slog.LevelError
		case status >= http.StatusBadRequest:
			level = slog.LevelWarn
		}
		logger.LogAttrs(c.Request.Context(), level, "http request", attrs...)
	}
}
