package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/reqctx"
	"github.com/karansnarula/personal-dashboard-api/internal/api/response"
	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

type DashboardService interface {
	Build(ctx context.Context, userID int64) (domain.Dashboard, error)
}

type Dashboard struct {
	svc DashboardService
}

func NewDashboard(svc DashboardService) *Dashboard { return &Dashboard{svc: svc} }

// Get returns the caller's widgets with live data and timing.
func (h *Dashboard) Get(c *gin.Context) {
	d, err := h.svc.Build(c.Request.Context(), reqctx.UserID(c))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, d)
}
