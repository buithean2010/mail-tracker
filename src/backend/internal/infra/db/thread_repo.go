package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type ThreadRepo struct {
	pool *pgxpool.Pool
}

func NewThreadRepo(pool *pgxpool.Pool) *ThreadRepo {
	return &ThreadRepo{pool: pool}
}

func (r *ThreadRepo) Upsert(ctx context.Context, t *domain.EmailThread) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO email_threads
		  (user_id, conversation_id, subject, from_address, received_at, last_message_at, message_count, raw_messages, summary_status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)
		ON CONFLICT (user_id, conversation_id) DO UPDATE
		SET subject = EXCLUDED.subject,
		    last_message_at = EXCLUDED.last_message_at,
		    message_count = EXCLUDED.message_count,
		    raw_messages = EXCLUDED.raw_messages,
		    summary_status = CASE
		        WHEN email_threads.summary_status IN ('done','processing') THEN email_threads.summary_status
		        ELSE EXCLUDED.summary_status
		    END,
		    updated_at = NOW()`,
		t.UserID, t.ConversationID, t.Subject, t.FromAddress,
		t.ReceivedAt, t.LastMessageAt, t.MessageCount, t.RawMessages, t.SummaryStatus)
	return err
}

func (r *ThreadRepo) ListByUser(ctx context.Context, userID uuid.UUID, f domain.ThreadFilter) ([]*domain.EmailThread, int, error) {
	args := []any{userID}
	conds := []string{"user_id = $1"}
	idx := 2

	if f.Status != "" {
		conds = append(conds, fmt.Sprintf("status = $%d", idx))
		args = append(args, f.Status)
		idx++
	}
	if f.Priority != "" {
		conds = append(conds, fmt.Sprintf("priority = $%d", idx))
		args = append(args, f.Priority)
		idx++
	}
	if f.From != "" {
		conds = append(conds, fmt.Sprintf("from_address ILIKE $%d", idx))
		args = append(args, "%"+f.From+"%")
		idx++
	}
	if f.Search != "" {
		conds = append(conds, fmt.Sprintf("(subject ILIKE $%d OR summary ILIKE $%d)", idx, idx))
		args = append(args, "%"+f.Search+"%")
		idx++
	}
	if f.DateFrom != nil {
		conds = append(conds, fmt.Sprintf("received_at >= $%d", idx))
		args = append(args, *f.DateFrom)
		idx++
	}
	if f.DateTo != nil {
		conds = append(conds, fmt.Sprintf("received_at <= $%d", idx))
		args = append(args, *f.DateTo)
		idx++
	}

	where := strings.Join(conds, " AND ")

	var total int
	if err := r.pool.QueryRow(ctx, "SELECT COUNT(*) FROM email_threads WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	limit := f.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := (f.Page - 1) * limit
	if offset < 0 {
		offset = 0
	}

	args = append(args, limit, offset)
	rows, err := r.pool.Query(ctx, fmt.Sprintf(`
		SELECT id, user_id, conversation_id, subject, from_address, received_at, last_message_at,
		       message_count, raw_messages, summary, priority, action_required, action_detail,
		       status, notes, summary_status, created_at, updated_at
		FROM email_threads WHERE %s
		ORDER BY received_at DESC
		LIMIT $%d OFFSET $%d`, where, idx, idx+1), args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var threads []*domain.EmailThread
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, 0, err
		}
		threads = append(threads, t)
	}
	return threads, total, rows.Err()
}

func (r *ThreadRepo) GetByID(ctx context.Context, id uuid.UUID) (*domain.EmailThread, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, user_id, conversation_id, subject, from_address, received_at, last_message_at,
		       message_count, raw_messages, summary, priority, action_required, action_detail,
		       status, notes, summary_status, created_at, updated_at
		FROM email_threads WHERE id = $1`, id)
	t, err := scanThread(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	return t, err
}

func (r *ThreadRepo) Update(ctx context.Context, t *domain.EmailThread) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE email_threads SET status=$1, notes=$2, updated_at=NOW() WHERE id=$3`,
		t.Status, t.Notes, t.ID)
	return err
}

func (r *ThreadRepo) GetPendingSummaries(ctx context.Context) ([]*domain.EmailThread, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, user_id, conversation_id, subject, from_address, received_at, last_message_at,
		       message_count, raw_messages, summary, priority, action_required, action_detail,
		       status, notes, summary_status, created_at, updated_at
		FROM email_threads WHERE summary_status = 'pending'
		ORDER BY received_at DESC LIMIT 100`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var threads []*domain.EmailThread
	for rows.Next() {
		t, err := scanThread(rows)
		if err != nil {
			return nil, err
		}
		threads = append(threads, t)
	}
	return threads, rows.Err()
}

func (r *ThreadRepo) UpdateSummaryStatus(ctx context.Context, id uuid.UUID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE email_threads SET summary_status=$1, updated_at=NOW() WHERE id=$2`,
		status, id)
	return err
}

func (r *ThreadRepo) UpdateSummary(ctx context.Context, id uuid.UUID, res *domain.SummaryResult) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE email_threads
		SET summary=$1, priority=$2, action_required=$3, action_detail=$4, summary_status='done', updated_at=NOW()
		WHERE id=$5`,
		res.Summary, res.Priority, res.ActionRequired, res.ActionDetail, id)
	return err
}

type scanner interface {
	Scan(dest ...any) error
}

func scanThread(s scanner) (*domain.EmailThread, error) {
	var t domain.EmailThread
	err := s.Scan(
		&t.ID, &t.UserID, &t.ConversationID, &t.Subject, &t.FromAddress,
		&t.ReceivedAt, &t.LastMessageAt, &t.MessageCount, &t.RawMessages,
		&t.Summary, &t.Priority, &t.ActionRequired, &t.ActionDetail,
		&t.Status, &t.Notes, &t.SummaryStatus, &t.CreatedAt, &t.UpdatedAt,
	)
	return &t, err
}
