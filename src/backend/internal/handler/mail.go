package handler

import (
	"context"
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
)

type jobStatus struct {
	Status string `json:"status"` // "running" | "done" | "failed"
	Error  string `json:"error,omitempty"`
}

type MailHandler struct {
	syncUC *app.SyncUseCase
	jobs   sync.Map // uuid.UUID → *jobStatus
}

func newMailHandler(syncUC *app.SyncUseCase) *MailHandler {
	return &MailHandler{syncUC: syncUC}
}

// POST /api/mail/sync
// Triggers a background mail sync for the current user and returns a job ID.
func (h *MailHandler) TriggerSync(c *gin.Context) {
	user := userFromCtx(c)
	jobID := uuid.New()
	status := &jobStatus{Status: "running"}
	h.jobs.Store(jobID, status)

	go func() {
		if err := h.syncUC.SyncUser(context.Background(), user.ID); err != nil {
			status.Status = "failed"
			status.Error = err.Error()
		} else {
			status.Status = "done"
		}
	}()

	c.JSON(http.StatusAccepted, gin.H{"job_id": jobID})
}

// GET /api/mail/sync/:job_id
// Returns the status of a previously triggered sync job.
func (h *MailHandler) GetSyncStatus(c *gin.Context) {
	jobID, err := uuid.Parse(c.Param("job_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid job id"})
		return
	}

	val, ok := h.jobs.Load(jobID)
	if !ok {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		return
	}

	c.JSON(http.StatusOK, val)
}
