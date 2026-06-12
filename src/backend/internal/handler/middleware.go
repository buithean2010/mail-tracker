package handler

import (
	"log"
	"net/http"

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

