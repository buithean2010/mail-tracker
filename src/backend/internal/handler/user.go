package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type UserHandler struct {
	userUC *app.UserUseCase
}

func newUserHandler(uc *app.UserUseCase) *UserHandler {
	return &UserHandler{userUC: uc}
}

// GET /api/users/me
func (h *UserHandler) GetMe(c *gin.Context) {
	user := userFromCtx(c)

	profile, err := h.userUC.GetProfile(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, profile)
}

// PATCH /api/users/me
func (h *UserHandler) UpdateMe(c *gin.Context) {
	user := userFromCtx(c)

	var body struct {
		DisplayName string `json:"display_name"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if err := h.userUC.UpdateDisplayName(c.Request.Context(), user.ID, body.DisplayName); err != nil {
		if errors.Is(err, domain.ErrInvalidInput) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "display_name required"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /api/users/me/filter
func (h *UserHandler) GetFilter(c *gin.Context) {
	user := userFromCtx(c)

	filter, err := h.userUC.GetFilter(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, filter)
}

// PATCH /api/users/me/filter
func (h *UserHandler) UpdateFilter(c *gin.Context) {
	user := userFromCtx(c)

	var settings domain.FilterSettings
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if err := h.userUC.UpdateFilter(c.Request.Context(), user.ID, &settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
