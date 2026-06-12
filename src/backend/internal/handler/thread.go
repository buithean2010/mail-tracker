package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type ThreadHandler struct {
	threadUC *app.ThreadUseCase
}

func newThreadHandler(uc *app.ThreadUseCase) *ThreadHandler {
	return &ThreadHandler{threadUC: uc}
}

// GET /api/threads
func (h *ThreadHandler) List(c *gin.Context) {
	user := userFromCtx(c)

	filter := domain.ThreadFilter{}
	filter.Status = c.Query("status")
	filter.Priority = c.Query("priority")
	filter.From = c.Query("from")
	filter.Search = c.Query("search")

	if v := c.Query("date_from"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.DateFrom = &t
		}
	}
	if v := c.Query("date_to"); v != "" {
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			filter.DateTo = &t
		}
	}

	filter.Page, _ = strconv.Atoi(c.DefaultQuery("page", "1"))
	filter.Limit, _ = strconv.Atoi(c.DefaultQuery("limit", "50"))

	threads, total, err := h.threadUC.List(c.Request.Context(), user.ID, filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"threads": threads,
		"total":   total,
		"page":    filter.Page,
		"limit":   filter.Limit,
	})
}

// PATCH /api/threads/:id
func (h *ThreadHandler) Update(c *gin.Context) {
	user := userFromCtx(c)

	threadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread id"})
		return
	}

	var body struct {
		Status string `json:"status"`
		Notes  string `json:"notes"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid body"})
		return
	}

	if err := h.threadUC.Update(c.Request.Context(), user.ID, threadID, body.Status, body.Notes); err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// POST /api/threads/:id/resummary
func (h *ThreadHandler) Resummary(c *gin.Context) {
	user := userFromCtx(c)

	threadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread id"})
		return
	}

	if err := h.threadUC.RequestResummary(c.Request.Context(), user.ID, threadID); err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
			return
		}
		if errors.Is(err, domain.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"ok": true})
}
