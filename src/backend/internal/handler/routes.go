package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

// Deps groups all dependencies needed to wire up handlers.
type Deps struct {
	AuthUC      *app.AuthUseCase
	SyncUC      *app.SyncUseCase
	SummaryUC   *app.SummaryUseCase
	ThreadUC    *app.ThreadUseCase
	UserUC      *app.UserUseCase
	SessionRepo domain.SessionRepo
	UserRepo    domain.UserRepo
	TokenRepo   domain.TokenRepo
	Broker      domain.SSEBroker
	Enc         domain.Encryptor
	Secure      bool   // false in dev, true in prod
	FrontendURL string // for CORS
}

func RegisterRoutes(r *gin.Engine, d Deps) {
	r.Use(CORSMiddleware(d.FrontendURL))

	authH := newAuthHandler(d.AuthUC, d.Secure)

	// Auth routes (public)
	auth := r.Group("/auth")
	{
		auth.GET("/login", authH.Login)
		auth.GET("/callback", authH.Callback)
		auth.POST("/logout", authH.Logout)
	}

	// Healthcheck (public)
	r.GET("/api/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"ok": true})
	})

	// Protected API routes
	api := r.Group("/api")
	api.Use(SessionMiddleware(d.SessionRepo, d.UserRepo, d.TokenRepo, d.AuthUC))
	{
		// Full route registration in P07
		api.GET("/healthz/auth", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})
	}
}
