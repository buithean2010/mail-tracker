package db_test

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
	"github.com/buithean2010/mail-tracker/backend/internal/infra/db"
)

// requireDB skips the test if TEST_DB_URL is not set.
// Run integration tests with: TEST_DB_URL=postgres://... go test ./internal/infra/db/...
func requireDB(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("TEST_DB_URL")
	if dsn == "" {
		t.Skip("TEST_DB_URL not set — skipping integration test")
	}
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		t.Fatalf("connect to test DB: %v", err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func seedUser(t *testing.T, pool *pgxpool.Pool) *domain.User {
	t.Helper()
	u := &domain.User{
		ID:             uuid.New(),
		MicrosoftID:    uuid.New().String(),
		Email:          "test-" + uuid.New().String() + "@example.com",
		DisplayName:    "Test User",
		FilterSettings: json.RawMessage(`{}`),
	}
	repo := db.NewUserRepo(pool)
	if err := repo.Upsert(context.Background(), u); err != nil {
		t.Fatalf("seed user: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM users WHERE id = $1", u.ID)
	})
	return u
}

func TestThreadRepo_UpsertAndGetByID(t *testing.T) {
	pool := requireDB(t)
	user := seedUser(t, pool)
	repo := db.NewThreadRepo(pool)

	thread := &domain.EmailThread{
		ID:             uuid.New(),
		UserID:         user.ID,
		ConversationID: "conv-" + uuid.New().String(),
		Subject:        "Integration test thread",
		FromAddress:    "sender@example.com",
		ReceivedAt:     time.Now().UTC().Truncate(time.Millisecond),
		LastMessageAt:  time.Now().UTC().Truncate(time.Millisecond),
		MessageCount:   1,
		RawMessages:    json.RawMessage(`[]`),
		Status:         "unread",
		SummaryStatus:  domain.SummaryStatusPending,
	}

	if err := repo.Upsert(context.Background(), thread); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM email_threads WHERE id = $1", thread.ID)
	})

	got, err := repo.GetByID(context.Background(), thread.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Subject != thread.Subject {
		t.Errorf("Subject = %q, want %q", got.Subject, thread.Subject)
	}
	if got.UserID != thread.UserID {
		t.Errorf("UserID mismatch")
	}
}

func TestThreadRepo_ListByUser(t *testing.T) {
	pool := requireDB(t)
	user := seedUser(t, pool)
	repo := db.NewThreadRepo(pool)

	for i := 0; i < 3; i++ {
		thread := &domain.EmailThread{
			ID:             uuid.New(),
			UserID:         user.ID,
			ConversationID: "conv-" + uuid.New().String(),
			Subject:        "Thread",
			FromAddress:    "s@example.com",
			ReceivedAt:     time.Now().UTC(),
			LastMessageAt:  time.Now().UTC(),
			MessageCount:   1,
			RawMessages:    json.RawMessage(`[]`),
			Status:         "unread",
			SummaryStatus:  domain.SummaryStatusPending,
		}
		if err := repo.Upsert(context.Background(), thread); err != nil {
			t.Fatalf("Upsert thread %d: %v", i, err)
		}
		tid := thread.ID
		t.Cleanup(func() {
			_, _ = pool.Exec(context.Background(), "DELETE FROM email_threads WHERE id = $1", tid)
		})
	}

	threads, total, err := repo.ListByUser(context.Background(), user.ID, domain.ThreadFilter{Limit: 50, Page: 1})
	if err != nil {
		t.Fatalf("ListByUser: %v", err)
	}
	if total < 3 {
		t.Errorf("total = %d, want >= 3", total)
	}
	if len(threads) < 3 {
		t.Errorf("len(threads) = %d, want >= 3", len(threads))
	}
}

func TestThreadRepo_UpdateSummaryStatus(t *testing.T) {
	pool := requireDB(t)
	user := seedUser(t, pool)
	repo := db.NewThreadRepo(pool)

	thread := &domain.EmailThread{
		ID:             uuid.New(),
		UserID:         user.ID,
		ConversationID: "conv-" + uuid.New().String(),
		Subject:        "Summary test",
		FromAddress:    "s@example.com",
		ReceivedAt:     time.Now().UTC(),
		LastMessageAt:  time.Now().UTC(),
		MessageCount:   1,
		RawMessages:    json.RawMessage(`[]`),
		Status:         "unread",
		SummaryStatus:  domain.SummaryStatusPending,
	}
	if err := repo.Upsert(context.Background(), thread); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), "DELETE FROM email_threads WHERE id = $1", thread.ID)
	})

	if err := repo.UpdateSummaryStatus(context.Background(), thread.ID, domain.SummaryStatusDone); err != nil {
		t.Fatalf("UpdateSummaryStatus: %v", err)
	}

	got, err := repo.GetByID(context.Background(), thread.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.SummaryStatus != domain.SummaryStatusDone {
		t.Errorf("SummaryStatus = %q, want done", got.SummaryStatus)
	}
}

func TestThreadRepo_GetByID_NotFound(t *testing.T) {
	pool := requireDB(t)
	repo := db.NewThreadRepo(pool)

	_, err := repo.GetByID(context.Background(), uuid.New())
	if err != domain.ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
