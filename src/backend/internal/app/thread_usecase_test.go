package app_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/app"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

// mockThreadRepo is an in-memory implementation of domain.ThreadRepo.
type mockThreadRepo struct {
	threads             map[uuid.UUID]*domain.EmailThread
	updateCalled        bool
	summaryStatusSaved  string
}

func newMockThreadRepo() *mockThreadRepo {
	return &mockThreadRepo{threads: make(map[uuid.UUID]*domain.EmailThread)}
}

func (m *mockThreadRepo) Upsert(_ context.Context, t *domain.EmailThread) error {
	m.threads[t.ID] = t
	return nil
}

func (m *mockThreadRepo) ListByUser(_ context.Context, userID uuid.UUID, _ domain.ThreadFilter) ([]*domain.EmailThread, int, error) {
	var result []*domain.EmailThread
	for _, t := range m.threads {
		if t.UserID == userID {
			result = append(result, t)
		}
	}
	return result, len(result), nil
}

func (m *mockThreadRepo) GetByID(_ context.Context, id uuid.UUID) (*domain.EmailThread, error) {
	t, ok := m.threads[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return t, nil
}

func (m *mockThreadRepo) Update(_ context.Context, t *domain.EmailThread) error {
	m.updateCalled = true
	m.threads[t.ID] = t
	return nil
}

func (m *mockThreadRepo) GetPendingSummaries(_ context.Context) ([]*domain.EmailThread, error) {
	return nil, nil
}

func (m *mockThreadRepo) UpdateSummaryStatus(_ context.Context, id uuid.UUID, status string) error {
	m.summaryStatusSaved = status
	if t, ok := m.threads[id]; ok {
		t.SummaryStatus = status
	}
	return nil
}

func (m *mockThreadRepo) UpdateSummary(_ context.Context, _ uuid.UUID, _ *domain.SummaryResult) error {
	return nil
}

// --- ThreadUseCase tests ---

func TestThreadUseCase_List_DefaultPagination(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	userID := uuid.New()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: userID, Subject: "Test thread"}

	threads, total, err := uc.List(context.Background(), userID, domain.ThreadFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(threads) != 1 {
		t.Errorf("len(threads) = %d, want 1", len(threads))
	}
}

func TestThreadUseCase_List_FiltersOtherUsers(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	userID := uuid.New()
	otherID := uuid.New()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: otherID}

	_, total, err := uc.List(context.Background(), userID, domain.ThreadFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 {
		t.Errorf("expected 0 threads for different user, got %d", total)
	}
}

func TestThreadUseCase_Update_Owner(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	ownerID := uuid.New()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: ownerID, Status: "unread", Notes: ""}

	if err := uc.Update(context.Background(), ownerID, tid, "read", "my note"); err != nil {
		t.Fatalf("expected no error from owner update: %v", err)
	}
	if !repo.updateCalled {
		t.Error("expected Update to be called on repo")
	}
	if repo.threads[tid].Status != "read" {
		t.Errorf("status = %q, want read", repo.threads[tid].Status)
	}
	if repo.threads[tid].Notes != "my note" {
		t.Errorf("notes = %q, want my note", repo.threads[tid].Notes)
	}
}

func TestThreadUseCase_Update_Unauthorized(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	ownerID := uuid.New()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: ownerID}

	err := uc.Update(context.Background(), uuid.New(), tid, "read", "")
	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestThreadUseCase_Update_NotFound(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	err := uc.Update(context.Background(), uuid.New(), uuid.New(), "read", "")
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestThreadUseCase_Update_PartialFields(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	ownerID := uuid.New()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: ownerID, Status: "unread", Notes: "keep this"}

	// Update only status — notes should remain unchanged.
	if err := uc.Update(context.Background(), ownerID, tid, "done", ""); err != nil {
		t.Fatal(err)
	}
	if repo.threads[tid].Notes != "keep this" {
		t.Errorf("notes was overwritten, got %q", repo.threads[tid].Notes)
	}
}

func TestThreadUseCase_RequestResummary_Owner(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	ownerID := uuid.New()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: ownerID, SummaryStatus: "done"}

	if err := uc.RequestResummary(context.Background(), ownerID, tid); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.summaryStatusSaved != domain.SummaryStatusPending {
		t.Errorf("summaryStatus = %q, want pending", repo.summaryStatusSaved)
	}
}

func TestThreadUseCase_RequestResummary_Unauthorized(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	ownerID := uuid.New()
	tid := uuid.New()
	repo.threads[tid] = &domain.EmailThread{ID: tid, UserID: ownerID}

	err := uc.RequestResummary(context.Background(), uuid.New(), tid)
	if err != domain.ErrUnauthorized {
		t.Errorf("expected ErrUnauthorized, got %v", err)
	}
}

func TestThreadUseCase_RequestResummary_NotFound(t *testing.T) {
	repo := newMockThreadRepo()
	uc := app.NewThreadUseCase(repo)

	err := uc.RequestResummary(context.Background(), uuid.New(), uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
