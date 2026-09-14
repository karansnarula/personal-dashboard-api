package service

import (
	"context"
	"encoding/json"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

type WidgetRepository interface {
	Create(ctx context.Context, userID int64, t domain.WidgetType, config json.RawMessage) (domain.Widget, error)
	ListByUser(ctx context.Context, userID int64) ([]domain.Widget, error)
	GetByID(ctx context.Context, id, userID int64) (domain.Widget, error)
	UpdateConfig(ctx context.Context, id, userID int64, config json.RawMessage) (domain.Widget, error)
	Delete(ctx context.Context, id, userID int64) error
}

type WidgetService struct {
	widgets WidgetRepository
}

func NewWidgetService(widgets WidgetRepository) *WidgetService {
	return &WidgetService{widgets: widgets}
}

// Create validates the type and config, stores the normalized config.
func (s *WidgetService) Create(ctx context.Context, userID int64, rawType string, config json.RawMessage) (domain.Widget, error) {
	t, err := domain.ParseWidgetType(rawType)
	if err != nil {
		return domain.Widget{}, err
	}
	normalized, err := NormalizeConfig(t, config)
	if err != nil {
		return domain.Widget{}, err
	}
	return s.widgets.Create(ctx, userID, t, normalized)
}

func (s *WidgetService) List(ctx context.Context, userID int64) ([]domain.Widget, error) {
	return s.widgets.ListByUser(ctx, userID)
}

func (s *WidgetService) Get(ctx context.Context, userID, id int64) (domain.Widget, error) {
	return s.widgets.GetByID(ctx, id, userID)
}

// UpdateConfig keeps the widget's type fixed and validates the new config
// against it.
func (s *WidgetService) UpdateConfig(ctx context.Context, userID, id int64, config json.RawMessage) (domain.Widget, error) {
	existing, err := s.widgets.GetByID(ctx, id, userID)
	if err != nil {
		return domain.Widget{}, err
	}
	normalized, err := NormalizeConfig(existing.Type, config)
	if err != nil {
		return domain.Widget{}, err
	}
	return s.widgets.UpdateConfig(ctx, id, userID, normalized)
}

func (s *WidgetService) Delete(ctx context.Context, userID, id int64) error {
	return s.widgets.Delete(ctx, id, userID)
}
