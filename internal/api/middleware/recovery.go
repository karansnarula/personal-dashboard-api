package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/reqctx"
	"github.com/karansnarula/personal-dashboard-api/internal/api/response"
)

// Recovery turns a handler panic into a logged 500 instead of a dropped
// connection.
func Recovery(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("panic recovered",
					"request_id", reqctx.RequestID(c),
					"panic", r,
					"stack", string(debug.Stack()),
				)
				response.Error(c, http.StatusInternalServerError, response.CodeInternal, "internal server error")
			}
		}()
		c.Next()
	}
}
