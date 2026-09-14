package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/reqctx"
	"github.com/karansnarula/personal-dashboard-api/internal/api/request"
	"github.com/karansnarula/personal-dashboard-api/internal/api/response"
	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

type WidgetService interface {
	Create(ctx context.Context, userID int64, rawType string, config json.RawMessage) (domain.Widget, error)
	List(ctx context.Context, userID int64) ([]domain.Widget, error)
	Get(ctx context.Context, userID, id int64) (domain.Widget, error)
	UpdateConfig(ctx context.Context, userID, id int64, config json.RawMessage) (domain.Widget, error)
	Delete(ctx context.Context, userID, id int64) error
}

type Widget struct {
	svc WidgetService
}

func NewWidget(svc WidgetService) *Widget { return &Widget{svc: svc} }

func (h *Widget) Create(c *gin.Context) {
	var req request.CreateWidget
	if !bindJSON(c, &req) {
		return
	}
	w, err := h.svc.Create(c.Request.Context(), reqctx.UserID(c), req.Type, req.Config)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, w)
}

func (h *Widget) List(c *gin.Context) {
	list, err := h.svc.List(c.Request.Context(), reqctx.UserID(c))
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.WidgetList{Widgets: list})
}

func (h *Widget) Get(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	w, err := h.svc.Get(c.Request.Context(), reqctx.UserID(c), id)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, w)
}

func (h *Widget) Update(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	var req request.UpdateWidget
	if !bindJSON(c, &req) {
		return
	}
	w, err := h.svc.UpdateConfig(c.Request.Context(), reqctx.UserID(c), id, req.Config)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, w)
}

func (h *Widget) Delete(c *gin.Context) {
	id, ok := pathID(c)
	if !ok {
		return
	}
	if err := h.svc.Delete(c.Request.Context(), reqctx.UserID(c), id); err != nil {
		response.FromError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
