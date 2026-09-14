package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"

	"github.com/karansnarula/personal-dashboard-api/internal/api/handler"
	"github.com/karansnarula/personal-dashboard-api/internal/api/middleware"
	"github.com/karansnarula/personal-dashboard-api/internal/api/response"
)

const maxBodyBytes = 1 << 20 // 1 MiB

// NewRouter builds the Gin engine with the global middleware chain and all
// route groups.
func NewRouter(cfg Config, deps Deps) *gin.Engine {
	if !cfg.Development {
		gin.SetMode(gin.ReleaseMode)
	}
	// Reject JSON bodies with fields we don't know about.
	binding.EnableDecoderDisallowUnknownFields = true

	r := gin.New()
	r.HandleMethodNotAllowed = true
	r.Use(
		middleware.RequestID(),
		middleware.Logger(deps.Logger),
		middleware.Recovery(deps.Logger),
		middleware.BodyLimit(maxBodyBytes),
	)
	r.NoRoute(func(c *gin.Context) {
		response.Error(c, http.StatusNotFound, response.CodeNotFound, "route not found")
	})
	r.NoMethod(func(c *gin.Context) {
		response.Error(c, http.StatusMethodNotAllowed, response.CodeMethodNotAllowed, "method not allowed")
	})

	health := handler.NewHealth(deps.DB)
	auth := handler.NewAuth(deps.AuthService)
	widgets := handler.NewWidget(deps.WidgetService)

	v1 := r.Group("/api/v1")
	v1.GET("/healthz", health.Check)

	v1.POST("/auth/register", auth.Register)
	v1.POST("/auth/login", auth.Login)
	v1.POST("/auth/logout", auth.Logout)

	protected := v1.Group("", middleware.Auth(deps.Tokens))
	{
		protected.POST("/widgets", widgets.Create)
		protected.GET("/widgets", widgets.List)
		protected.GET("/widgets/:id", widgets.Get)
		protected.PATCH("/widgets/:id", widgets.Update)
		protected.DELETE("/widgets/:id", widgets.Delete)
	}

	return r
}
