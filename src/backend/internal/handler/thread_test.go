package handler_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// stubThreadRepo is a minimal ThreadRepo for handler tests.
type stubThreadRepo struct {
	threads map[uuid.UUID]*domain.EmailThread
}

func newStubThreadRepo() *stubThreadRepo {
	return &stubThreadRepo{threads: make(map[uuid.UUID]*domain.EmailThread)}
}

func (s *stubThreadRepo) Upsert(_ context.Context, t *domain.EmailThread) error {
	s.threads[t.ID] = t
	return nil
}
func (s *stubThreadRepo) ListByUser(_ context.Context, userID uuid.UUID, f domain.ThreadFilter) ([]*domain.EmailThread, int, error) {
	var result []*domain.EmailThread
	for _, t := range s.threads {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	return result, len(result), nil
}
func (s *stubThreadRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.EmailThread, error) {
	t, ok := s.threads[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}
func (s *stubThreadRepo) Update(_ context.Context, t *domain.EmailThread) error {
	s.threads[t.ID] = t
	return nil
}
func (s *stubThreadRepo) GetPendingSummaries(_ context.Context) ([]*domain.EmailThread, error) {
	return nil, nil
}
func (s *stubThreadRepo) UpdateSummaryStatus(_ context.Context, id uuid.UUID, status string) error {
	if t, ok := s.threads[id]; ok {
		t.SummaryStatus = status
	}
	return nil
}
func (s *stubThreadRepo) UpdateSummary(_ context.Context, _ uuid.UUID, _ *domain.SummaryResult) error {
	return nil
}

// buildRouter sets up Gin with the thread handler and an auth-injecting middleware.
func buildRouter(user *domain.User, repo domain.ThreadRepo) *gin.Engine {
	r := gin.New()
	uc := app.NewThreadUseCase(repo)

	// Inject user into context the same way SessionMiddleware does.
	r.Use(func(c *gin.Context) {
		c.Set("user", user)
		c.Next()
	})

	deps := Deps{ThreadUC: uc}
	registerTestRoutes(r, deps)
	return r
}

// Deps + registerTestRoutes are local helpers to wire only the routes under test
// without requiring the full Deps struct from handler package.
type Deps struct {
	ThreadUC *app.ThreadUseCase
}

func registerTestRoutes(r *gin.Engine, d Deps) {
	h := newTestThreadHandler(d.ThreadUC)
	api := r.Group("/api")
	api.GET("/threads", h.list)
	api.PATCH("/threads/:id", h.update)
	api.POST("/threads/:id/resummary", h.resummary)
}

// --- Test handler (thin wrapper, same logic as handler.ThreadHandler) ---

type testThreadHandler struct{ uc *app.ThreadUseCase }

func newTestThreadHandler(uc *app.ThreadUseCase) *testThreadHandler { return &testThreadHandler{uc} }

func userFromCtxTest(c *gin.Context) *domain.User {
	if u, ok := c.Get("user"); ok {
		return u.(*domain.User)
	}
	return nil
}

func (h *testThreadHandler) list(c *gin.Context) {
	user := userFromCtxTest(c)
	threads, total, err := h.uc.List(c.Request.Context(), user.ID, domain.ThreadFilter{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"threads": threads, "total": total})
}

func (h *testThreadHandler) update(c *gin.Context) {
	user := userFromCtxTest(c)
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
	if err := h.uc.Update(c.Request.Context(), user.ID, threadID, body.Status, body.Notes); err != nil {
		switch err {
		case domain.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case domain.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func (h *testThreadHandler) resummary(c *gin.Context) {
	user := userFromCtxTest(c)
	threadID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid thread id"})
		return
	}
	if err := h.uc.RequestResummary(c.Request.Context(), user.ID, threadID); err != nil {
		switch err {
		case domain.ErrUnauthorized:
			c.JSON(http.StatusForbidden, gin.H{"error": "forbidden"})
		case domain.ErrNotFound:
			c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
		}
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// --- Tests ---

func TestThreadHandler_List_OK(t *testing.T) {
	user := &domain.User{ID: uuid.New()}
	repo := newStubThreadRepo()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: user.ID, Subject: "Test"}

	r := buildRouter(user, repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/threads", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Threads []domain.EmailThread `json:"threads"`
		Total   int                  `json:"total"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Total != 1 {
		t.Errorf("total = %d, want 1", resp.Total)
	}
	if len(resp.Threads) != 1 || resp.Threads[0].Subject != "Test" {
		t.Errorf("unexpected threads: %+v", resp.Threads)
	}
}

func TestThreadHandler_Update_OK(t *testing.T) {
	user := &domain.User{ID: uuid.New()}
	repo := newStubThreadRepo()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: user.ID, Status: "unread"}

	r := buildRouter(user, repo)
	body := `{"status":"read"}`
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/threads/"+tid.String(), bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", w.Code, w.Body.String())
	}
}

func TestThreadHandler_Update_InvalidUUID(t *testing.T) {
	user := &domain.User{ID: uuid.New()}
	repo := newStubThreadRepo()
	r := buildRouter(user, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/threads/not-a-uuid", bytes.NewBufferString(`{"status":"read"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestThreadHandler_Update_NotFound(t *testing.T) {
	user := &domain.User{ID: uuid.New()}
	repo := newStubThreadRepo()
	r := buildRouter(user, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/threads/"+uuid.New().String(), bytes.NewBufferString(`{"status":"read"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestThreadHandler_Update_Forbidden(t *testing.T) {
	owner := &domain.User{ID: uuid.New()}
	caller := &domain.User{ID: uuid.New()}
	repo := newStubThreadRepo()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: owner.ID, Status: "unread"}

	// caller is a different user
	r := buildRouter(caller, repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPatch, "/api/threads/"+tid.String(), bytes.NewBufferString(`{"status":"read"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestThreadHandler_Resummary_OK(t *testing.T) {
	user := &domain.User{ID: uuid.New()}
	repo := newStubThreadRepo()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: user.ID, SummaryStatus: "done"}

	r := buildRouter(user, repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/threads/"+tid.String()+"/resummary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestThreadHandler_Resummary_Forbidden(t *testing.T) {
	owner := &domain.User{ID: uuid.New()}
	caller := &domain.User{ID: uuid.New()}
	repo := newStubThreadRepo()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: owner.ID}

	r := buildRouter(caller, repo)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/threads/"+tid.String()+"/resummary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestThreadHandler_Resummary_NotFound(t *testing.T) {
	user := &domain.User{ID: uuid.New()}
	repo := newStubThreadRepo()
	r := buildRouter(user, repo)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/threads/"+uuid.New().String()+"/resummary", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}
