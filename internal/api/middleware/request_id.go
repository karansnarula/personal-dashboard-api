package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/karansnarula/personal-dashboard-api/internal/api/reqctx"
)

const (
	HeaderRequestID    = "X-Request-ID"
	maxRequestIDLength = 128
)

// RequestID honours an incoming X-Request-ID (so callers can correlate
// logs) or generates one, and echoes it back on the response.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" || len(id) > maxRequestIDLength {
			id = uuid.NewString()
		}
		reqctx.SetRequestID(c, id)
		c.Header(HeaderRequestID, id)
		c.Next()
	}
}
