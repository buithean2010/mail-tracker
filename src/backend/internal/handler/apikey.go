package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

// apiKeyBody is the request shape for POST /api/users/me/api-key and /test.
// domain.UserAPIKey uses json:"-" for sensitive fields so we use a separate struct.
type apiKeyBody struct {
	AIMode       string `json:"ai_mode"`
	Provider     string `json:"provider"`
	APIKey       string `json:"api_key"`
	Model        string `json:"model"`
	PAWebhookURL string `json:"pa_webhook_url"`
}

type APIKeyHandler struct {
	userUC    *app.UserUseCase
	summaryUC *app.SummaryUseCase
}

func newAPIKeyHandler(userUC *app.UserUseCase, summaryUC *app.SummaryUseCase) *APIKeyHandler {
	return &APIKeyHandler{userUC: userUC, summaryUC: summaryUC}
}

// POST /api/users/me/api-key
func (h *APIKeyHandler) Upsert(c *gin.Context) {
	user := userFromCtx(c)

	var body apiKeyBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	switch body.AIMode {
	case domain.AIModeByok:
		if body.APIKey == "" || body.Provider == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "api_key and provider required for byok"})
			return
		}
	case domain.AIModePA:
		if body.PAWebhookURL == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "pa_webhook_url required for power_automate"})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "ai_mode must be byok or power_automate"})
		return
	}

	key := &domain.UserAPIKey{
		AIMode:       body.AIMode,
		Provider:     body.Provider,
		APIKey:       body.APIKey,
		Model:        body.Model,
		PAWebhookURL: body.PAWebhookURL,
	}

	if err := h.userUC.UpsertAPIKey(c.Request.Context(), user.ID, key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DELETE /api/users/me/api-key
func (h *APIKeyHandler) Delete(c *gin.Context) {
	user := userFromCtx(c)

	if err := h.userUC.DeleteAPIKey(c.Request.Context(), user.ID); err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/users/me/api-key/test
// Runs a sample AI call with the provided key without saving it.
func (h *APIKeyHandler) Test(c *gin.Context) {
	var body apiKeyBody
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	key := &domain.UserAPIKey{
		AIMode:       body.AIMode,
		Provider:     body.Provider,
		APIKey:       body.APIKey,
		Model:        body.Model,
		PAWebhookURL: body.PAWebhookURL,
	}

	result, err := h.summaryUC.TestKey(c.Request.Context(), key)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}
