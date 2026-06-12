package handler

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

const (
	sessionCookieName = "session_id"
	stateCookieName   = "oauth_state"
	verifierCookieName = "pkce_verifier"
	cookiePath        = "/"
)

type AuthHandler struct {
	authUC *app.AuthUseCase
	secure bool // set true in production
}

func newAuthHandler(authUC *app.AuthUseCase, secure bool) *AuthHandler {
	return &AuthHandler{authUC: authUC, secure: secure}
}

// GET /auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	state, err := randomHex(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	verifier := oauth2.GenerateVerifier()

	// Store state + verifier in short-lived cookies (5 min TTL)
	maxAge := 5 * 60
	c.SetCookie(stateCookieName, state, maxAge, cookiePath, "", h.secure, true)
	c.SetCookie(verifierCookieName, verifier, maxAge, cookiePath, "", h.secure, true)

	authURL := h.authUC.BuildAuthURL(state, verifier)
	c.Redirect(http.StatusFound, authURL)
}

// GET /auth/callback
func (h *AuthHandler) Callback(c *gin.Context) {
	// Validate state
	storedState, err := c.Cookie(stateCookieName)
	if err != nil || storedState != c.Query("state") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid state"})
		return
	}

	verifier, err := c.Cookie(verifierCookieName)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing pkce verifier"})
		return
	}

	code := c.Query("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing code"})
		return
	}

	// Exchange code for tokens
	ut, userInfo, err := h.authUC.ExchangeCode(c.Request.Context(), code, verifier)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "auth failed"})
		return
	}

	// Upsert user + token
	user, err := h.authUC.UpsertUserAndToken(c.Request.Context(), userInfo, ut)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// Create session
	session, err := h.authUC.CreateSession(c.Request.Context(), user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	// Clear PKCE cookies
	c.SetCookie(stateCookieName, "", -1, cookiePath, "", h.secure, true)
	c.SetCookie(verifierCookieName, "", -1, cookiePath, "", h.secure, true)

	// Set session cookie: 7-day HttpOnly
	ttl := int(time.Until(session.ExpiresAt).Seconds())
	c.SetCookie(sessionCookieName, session.ID, ttl, cookiePath, "", h.secure, true)

	c.Redirect(http.StatusFound, "/")
}

// POST /auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	sessionID, err := c.Cookie(sessionCookieName)
	if err == nil && sessionID != "" {
		_ = h.authUC.DeleteSession(c.Request.Context(), sessionID)
	}

	c.SetCookie(sessionCookieName, "", -1, cookiePath, "", h.secure, true)
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// contextKey is unexported to avoid collisions
type contextKey string

const userKey contextKey = "user"

func userFromCtx(c *gin.Context) *domain.User {
	if u, ok := c.Get(string(userKey)); ok {
		return u.(*domain.User)
	}
	return nil
}
