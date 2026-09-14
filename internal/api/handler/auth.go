package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/karansnarula/personal-dashboard-api/internal/api/request"
	"github.com/karansnarula/personal-dashboard-api/internal/api/response"
	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

type AuthService interface {
	Register(ctx context.Context, email, password string) (domain.User, error)
	Login(ctx context.Context, email, password string) (domain.AuthToken, error)
}

type Auth struct {
	svc AuthService
}

func NewAuth(svc AuthService) *Auth { return &Auth{svc: svc} }

func (h *Auth) Register(c *gin.Context) {
	var req request.Register
	if !bindJSON(c, &req) {
		return
	}
	user, err := h.svc.Register(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.JSON(c, http.StatusCreated, response.Registered{ID: user.ID, Email: user.Email})
}

func (h *Auth) Login(c *gin.Context) {
	var req request.Login
	if !bindJSON(c, &req) {
		return
	}
	tok, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		response.FromError(c, err)
		return
	}
	response.JSON(c, http.StatusOK, response.Token{
		Token:     tok.Token,
		TokenType: "Bearer",
		ExpiresIn: int64(tok.ExpiresIn.Seconds()),
	})
}

// Logout exists for API completeness. Tokens are stateless JWTs, so logging
// out means the client discards its token.
func (h *Auth) Logout(c *gin.Context) {
	response.JSON(c, http.StatusOK, response.Message{Message: "logged out; discard the token on the client"})
}
