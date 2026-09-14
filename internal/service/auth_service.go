package service

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/karansnarula/personal-dashboard-api/internal/auth"
	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

const (
	minPasswordLength = 8
	maxEmailLength    = 254
)

// dummyHash is compared against when the email is unknown, so login takes
// the same time whether or not the account exists.
var dummyHash, _ = auth.HashPassword("dummy-password-for-timing")

type UserRepository interface {
	Create(ctx context.Context, email, passwordHash string) (domain.User, error)
	GetByEmail(ctx context.Context, email string) (domain.User, error)
}

type TokenIssuer interface {
	Issue(userID int64) (string, error)
	TTL() time.Duration
}

type AuthService struct {
	users  UserRepository
	tokens TokenIssuer
}

func NewAuthService(users UserRepository, tokens TokenIssuer) *AuthService {
	return &AuthService{users: users, tokens: tokens}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (domain.User, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return domain.User{}, err
	}
	if err := validatePassword(password); err != nil {
		return domain.User{}, err
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return domain.User{}, fmt.Errorf("hash password: %w", err)
	}
	return s.users.Create(ctx, email, hash)
}

// Login returns ErrInvalidCredentials for both unknown emails and wrong
// passwords so the endpoint does not reveal which emails are registered.
func (s *AuthService) Login(ctx context.Context, email, password string) (domain.AuthToken, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	user, err := s.users.GetByEmail(ctx, email)
	if errors.Is(err, domain.ErrNotFound) {
		auth.CheckPassword(dummyHash, password)
		return domain.AuthToken{}, domain.ErrInvalidCredentials
	}
	if err != nil {
		return domain.AuthToken{}, err
	}
	if !auth.CheckPassword(user.PasswordHash, password) {
		return domain.AuthToken{}, domain.ErrInvalidCredentials
	}

	token, err := s.tokens.Issue(user.ID)
	if err != nil {
		return domain.AuthToken{}, fmt.Errorf("issue token: %w", err)
	}
	return domain.AuthToken{Token: token, ExpiresIn: s.tokens.TTL()}, nil
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", domain.Validationf("email is required")
	}
	if len(email) > maxEmailLength {
		return "", domain.Validationf("email must be at most %d characters", maxEmailLength)
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email {
		return "", domain.Validationf("email %q is not a valid address", raw)
	}
	return email, nil
}

func validatePassword(password string) error {
	if len(password) < minPasswordLength {
		return domain.Validationf("password must be at least %d characters", minPasswordLength)
	}
	if len(password) > auth.MaxPasswordLength {
		return domain.Validationf("password must be at most %d bytes", auth.MaxPasswordLength)
	}
	return nil
}
