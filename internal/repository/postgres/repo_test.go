package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/karansnarula/personal-dashboard-api/internal/database"
	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

// These are integration tests against a real Postgres. They are skipped
// unless TEST_DATABASE_URL is set, e.g.
//
//	TEST_DATABASE_URL=postgres://dashboard:dashboard@localhost:5432/dashboard?sslmode=disable go test ./internal/repository/...
func testPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set; skipping Postgres integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := database.NewPool(ctx, url)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := database.Migrate(ctx, pool, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// Each test starts from empty tables. widgets cascades from users.
	if _, err := pool.Exec(ctx, `TRUNCATE users RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
	return pool
}

func TestUserRepository(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	repo := NewUserRepository(pool)

	u, err := repo.Create(ctx, "a@example.com", "hash")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if u.ID == 0 || u.Email != "a@example.com" || u.PasswordHash != "hash" || u.CreatedAt.IsZero() {
		t.Fatalf("unexpected user: %+v", u)
	}

	if _, err := repo.Create(ctx, "a@example.com", "other"); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("duplicate email: got %v, want ErrEmailTaken", err)
	}

	got, err := repo.GetByEmail(ctx, "a@example.com")
	if err != nil || got.ID != u.ID {
		t.Fatalf("get by email: %+v, %v", got, err)
	}
	if _, err := repo.GetByEmail(ctx, "missing@example.com"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("missing email: got %v, want ErrNotFound", err)
	}
}

func TestWidgetRepository(t *testing.T) {
	pool := testPool(t)
	ctx := context.Background()
	users := NewUserRepository(pool)
	repo := NewWidgetRepository(pool)

	owner, _ := users.Create(ctx, "owner@example.com", "h")
	other, _ := users.Create(ctx, "other@example.com", "h")

	empty, err := repo.ListByUser(ctx, owner.ID)
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty list: %v, %v", empty, err)
	}

	cfg := json.RawMessage(`{"city":"London"}`)
	w, err := repo.Create(ctx, owner.ID, domain.WidgetWeather, cfg)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if w.ID == 0 || w.UserID != owner.ID || w.Type != domain.WidgetWeather || w.CreatedAt.IsZero() {
		t.Fatalf("unexpected widget: %+v", w)
	}
	assertJSONEqual(t, w.Config, cfg)

	list, err := repo.ListByUser(ctx, owner.ID)
	if err != nil || len(list) != 1 || list[0].ID != w.ID {
		t.Fatalf("list: %+v, %v", list, err)
	}

	got, err := repo.GetByID(ctx, w.ID, owner.ID)
	if err != nil || got.ID != w.ID {
		t.Fatalf("get: %+v, %v", got, err)
	}
	if _, err := repo.GetByID(ctx, w.ID, other.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user get: got %v, want ErrNotFound", err)
	}

	newCfg := json.RawMessage(`{"city":"Paris"}`)
	updated, err := repo.UpdateConfig(ctx, w.ID, owner.ID, newCfg)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	assertJSONEqual(t, updated.Config, newCfg)
	if !updated.UpdatedAt.After(w.UpdatedAt) {
		t.Fatalf("updated_at not bumped: %v -> %v", w.UpdatedAt, updated.UpdatedAt)
	}
	if _, err := repo.UpdateConfig(ctx, w.ID, other.ID, newCfg); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user update: got %v, want ErrNotFound", err)
	}

	if err := repo.Delete(ctx, w.ID, other.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user delete: got %v, want ErrNotFound", err)
	}
	if err := repo.Delete(ctx, w.ID, owner.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := repo.Delete(ctx, w.ID, owner.ID); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("delete twice: got %v, want ErrNotFound", err)
	}
}

func assertJSONEqual(t *testing.T, got, want json.RawMessage) {
	t.Helper()
	var g, w any
	if err := json.Unmarshal(got, &g); err != nil {
		t.Fatalf("got is not JSON: %s", got)
	}
	if err := json.Unmarshal(want, &w); err != nil {
		t.Fatalf("want is not JSON: %s", want)
	}
	gb, _ := json.Marshal(g)
	wb, _ := json.Marshal(w)
	if string(gb) != string(wb) {
		t.Fatalf("config mismatch: got %s, want %s", gb, wb)
	}
}
