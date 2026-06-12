package handler

import (
	"fmt"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type SSEHandler struct {
	broker domain.SSEBroker
}

func newSSEHandler(broker domain.SSEBroker) *SSEHandler {
	return &SSEHandler{broker: broker}
}

// GET /api/events
// Upgrades the connection to SSE and streams events for the current user.
func (h *SSEHandler) Stream(c *gin.Context) {
	user := userFromCtx(c)

	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no") // disable nginx proxy buffering

	ch, cancel := h.broker.Subscribe(user.ID)
	defer cancel()

	c.Stream(func(w io.Writer) bool {
		select {
		case <-c.Request.Context().Done():
			return false
		case ev, ok := <-ch:
			if !ok {
				return false
			}
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Event, ev.Data)
			return true
		}
	})

	c.Status(http.StatusOK)
}
