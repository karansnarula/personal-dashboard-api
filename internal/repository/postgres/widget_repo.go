package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

type WidgetRepository struct {
	pool *pgxpool.Pool
}

func NewWidgetRepository(pool *pgxpool.Pool) *WidgetRepository {
	return &WidgetRepository{pool: pool}
}

const widgetColumns = `id, user_id, type, config, created_at, updated_at`

func scanWidget(row pgx.Row) (domain.Widget, error) {
	var (
		w      domain.Widget
		typ    string
		config []byte
	)
	if err := row.Scan(&w.ID, &w.UserID, &typ, &config, &w.CreatedAt, &w.UpdatedAt); err != nil {
		return domain.Widget{}, err
	}
	w.Type = domain.WidgetType(typ)
	w.Config = json.RawMessage(config)
	return w, nil
}

func (r *WidgetRepository) Create(ctx context.Context, userID int64, t domain.WidgetType, config json.RawMessage) (domain.Widget, error) {
	const q = `
		INSERT INTO widgets (user_id, type, config)
		VALUES ($1, $2, $3)
		RETURNING ` + widgetColumns

	w, err := scanWidget(r.pool.QueryRow(ctx, q, userID, string(t), string(config)))
	if err != nil {
		return domain.Widget{}, fmt.Errorf("insert widget: %w", err)
	}
	return w, nil
}

// ListByUser returns the user's widgets in creation order. It always
// returns a non-nil slice so the JSON encodes as [] rather than null.
func (r *WidgetRepository) ListByUser(ctx context.Context, userID int64) ([]domain.Widget, error) {
	const q = `
		SELECT ` + widgetColumns + `
		FROM widgets
		WHERE user_id = $1
		ORDER BY id`

	rows, err := r.pool.Query(ctx, q, userID)
	if err != nil {
		return nil, fmt.Errorf("list widgets: %w", err)
	}
	widgets, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (domain.Widget, error) {
		return scanWidget(row)
	})
	if err != nil {
		return nil, fmt.Errorf("scan widgets: %w", err)
	}
	if widgets == nil {
		widgets = []domain.Widget{}
	}
	return widgets, nil
}

// GetByID is scoped to the owner so users cannot read each other's widgets
// by guessing IDs; a foreign widget looks identical to a missing one.
func (r *WidgetRepository) GetByID(ctx context.Context, id, userID int64) (domain.Widget, error) {
	const q = `
		SELECT ` + widgetColumns + `
		FROM widgets
		WHERE id = $1 AND user_id = $2`

	w, err := scanWidget(r.pool.QueryRow(ctx, q, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Widget{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Widget{}, fmt.Errorf("select widget: %w", err)
	}
	return w, nil
}

func (r *WidgetRepository) UpdateConfig(ctx context.Context, id, userID int64, config json.RawMessage) (domain.Widget, error) {
	const q = `
		UPDATE widgets
		SET config = $1, updated_at = now()
		WHERE id = $2 AND user_id = $3
		RETURNING ` + widgetColumns

	w, err := scanWidget(r.pool.QueryRow(ctx, q, string(config), id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Widget{}, domain.ErrNotFound
	}
	if err != nil {
		return domain.Widget{}, fmt.Errorf("update widget: %w", err)
	}
	return w, nil
}

func (r *WidgetRepository) Delete(ctx context.Context, id, userID int64) error {
	const q = `DELETE FROM widgets WHERE id = $1 AND user_id = $2`

	tag, err := r.pool.Exec(ctx, q, id, userID)
	if err != nil {
		return fmt.Errorf("delete widget: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}
	return nil
}
