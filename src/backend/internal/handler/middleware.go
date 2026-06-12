package handler

import (
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

// SessionMiddleware loads the current user from the session cookie.
// If the session is missing or expired → 401.
// If the token is near expiry → refresh it automatically.
func SessionMiddleware(
	sessionRepo domain.SessionRepo,
	userRepo domain.UserRepo,
	tokenRepo domain.TokenRepo,
	authUC *app.AuthUseCase,
) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie(sessionCookieName)
		if err != nil || sessionID == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		session, err := sessionRepo.GetByID(c.Request.Context(), sessionID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		user, err := userRepo.GetByID(c.Request.Context(), session.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}

		if user.NeedsReauth {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "reauth_required"})
			return
		}

		// Auto-refresh token if < 5 minutes remaining
		token, err := tokenRepo.GetByUserID(c.Request.Context(), user.ID)
		if err == nil && token.NeedsRefresh() {
			newToken, refreshErr := authUC.RefreshToken(c.Request.Context(), user.ID, token.RefreshToken)
			if refreshErr != nil {
				log.Printf("token refresh failed for user %s: %v", user.ID, refreshErr)
				if err := setNeedsReauth(c, userRepo, user); err == nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "reauth_required"})
					return
				}
			} else {
				newToken.UserID = user.ID
				_ = tokenRepo.Upsert(c.Request.Context(), newToken)
			}
		}

		c.Set(string(userKey), user)
		c.Next()
	}
}

func setNeedsReauth(c *gin.Context, repo domain.UserRepo, user *domain.User) error {
	return repo.SetNeedsReauth(c.Request.Context(), user.ID, true)
}

// RateLimitMiddleware enforces a fixed-window rate limit per user (or IP for unauthenticated routes).
// maxReqs requests are allowed per windowSec seconds.
func RateLimitMiddleware(maxReqs int, windowSec int) gin.HandlerFunc {
	type window struct {
		count int64
		start int64 // unix seconds
	}
	var (
		mu      sync.Mutex
		buckets = make(map[string]*window)
	)
	max := int64(maxReqs)
	win := int64(windowSec)

	return func(c *gin.Context) {
		key := c.ClientIP()
		if u := userFromCtx(c); u != nil {
			key = u.ID.String()
		}

		now := time.Now().Unix()
		mu.Lock()
		b, ok := buckets[key]
		if !ok || now-b.start >= win {
			buckets[key] = &window{count: 1, start: now}
			mu.Unlock()
			c.Next()
			return
		}
		if b.count >= max {
			mu.Unlock()
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		b.count++
		mu.Unlock()
		c.Next()
	}
}

// CORSMiddleware sets CORS headers for the frontend origin.
func CORSMiddleware(frontendOrigin string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", frontendOrigin)
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		c.Next()
	}
}

