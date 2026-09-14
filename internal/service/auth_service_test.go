package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

type fakeUserRepo struct {
	users  map[string]domain.User
	nextID int64
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]domain.User{}, nextID: 1}
}

func (r *fakeUserRepo) Create(_ context.Context, email, hash string) (domain.User, error) {
	if _, exists := r.users[email]; exists {
		return domain.User{}, domain.ErrEmailTaken
	}
	u := domain.User{ID: r.nextID, Email: email, PasswordHash: hash, CreatedAt: time.Now()}
	r.nextID++
	r.users[email] = u
	return u, nil
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (domain.User, error) {
	u, ok := r.users[email]
	if !ok {
		return domain.User{}, domain.ErrNotFound
	}
	return u, nil
}

type fakeTokens struct{}

func (fakeTokens) Issue(userID int64) (string, error) { return "token-for-user", nil }
func (fakeTokens) TTL() time.Duration                 { return time.Hour }

func TestAuthService_Register(t *testing.T) {
	ctx := context.Background()
	svc := NewAuthService(newFakeUserRepo(), fakeTokens{})

	u, err := svc.Register(ctx, "  Me@Example.COM ", "hunter2hunter2")
	if err != nil {
		t.Fatalf("register: %v", err)
	}
	if u.Email != "me@example.com" {
		t.Fatalf("email not normalized: %q", u.Email)
	}
	if u.PasswordHash == "" || u.PasswordHash == "hunter2hunter2" {
		t.Fatalf("password not hashed: %q", u.PasswordHash)
	}

	if _, err := svc.Register(ctx, "me@example.com", "hunter2hunter2"); !errors.Is(err, domain.ErrEmailTaken) {
		t.Fatalf("duplicate: got %v, want ErrEmailTaken", err)
	}

	for name, in := range map[string][2]string{
		"short password": {"a@b.com", "short"},
		"bad email":      {"not-an-email", "hunter2hunter2"},
		"empty email":    {"", "hunter2hunter2"},
		"display name":   {"Bob <bob@example.com>", "hunter2hunter2"},
	} {
		if _, err := svc.Register(ctx, in[0], in[1]); !domain.IsValidation(err) {
			t.Errorf("%s: got %v, want ValidationError", name, err)
		}
	}
}

func TestAuthService_Login(t *testing.T) {
	ctx := context.Background()
	svc := NewAuthService(newFakeUserRepo(), fakeTokens{})
	if _, err := svc.Register(ctx, "me@example.com", "hunter2hunter2"); err != nil {
		t.Fatal(err)
	}

	if _, err := svc.Login(ctx, "nobody@example.com", "hunter2hunter2"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("unknown email: got %v", err)
	}
	if _, err := svc.Login(ctx, "me@example.com", "wrong-password"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Fatalf("wrong password: got %v", err)
	}

	tok, err := svc.Login(ctx, "ME@example.com", "hunter2hunter2")
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	if tok.Token != "token-for-user" || tok.ExpiresIn != time.Hour {
		t.Fatalf("unexpected token: %+v", tok)
	}
}
