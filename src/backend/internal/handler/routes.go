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
	threadH := newThreadHandler(d.ThreadUC)
	userH := newUserHandler(d.UserUC)
	apiKeyH := newAPIKeyHandler(d.UserUC, d.SummaryUC)
	mailH := newMailHandler(d.SyncUC)
	sseH := newSSEHandler(d.Broker)

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
	api.Use(RateLimitMiddleware(120, 60)) // 120 req/min per user
	{
		// Threads
		api.GET("/threads", threadH.List)
		api.PATCH("/threads/:id", threadH.Update)
		api.POST("/threads/:id/resummary", threadH.Resummary)

		// User profile + filter
		api.GET("/users/me", userH.GetMe)
		api.PATCH("/users/me", userH.UpdateMe)
		api.GET("/users/me/filter", userH.GetFilter)
		api.PATCH("/users/me/filter", userH.UpdateFilter)

		// API key management
		api.POST("/users/me/api-key", apiKeyH.Upsert)
		api.DELETE("/users/me/api-key", apiKeyH.Delete)
		api.POST("/users/me/api-key/test", apiKeyH.Test)

		// Mail sync
		api.POST("/mail/sync", mailH.TriggerSync)
		api.GET("/mail/sync/:job_id", mailH.GetSyncStatus)

		// SSE stream
		api.GET("/events", sseH.Stream)

		// Auth check
		api.GET("/healthz/auth", func(c *gin.Context) {
			c.JSON(200, gin.H{"ok": true})
		})
	}
}
