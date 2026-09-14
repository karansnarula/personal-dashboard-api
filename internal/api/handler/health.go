package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/response"
)

const healthPingTimeout = 2 * time.Second

// Pinger is satisfied by *pgxpool.Pool.
type Pinger interface {
	Ping(ctx context.Context) error
}

type Health struct {
	db Pinger
}

func NewHealth(db Pinger) *Health { return &Health{db: db} }

// Check reports 200 when the database answers, 503 otherwise.
func (h *Health) Check(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), healthPingTimeout)
	defer cancel()

	if err := h.db.Ping(ctx); err != nil {
		_ = c.Error(err)
		response.Error(c, http.StatusServiceUnavailable, response.CodeUnavailable, "database unreachable")
		return
	}
	response.JSON(c, http.StatusOK, gin.H{"status": "ok"})
}
