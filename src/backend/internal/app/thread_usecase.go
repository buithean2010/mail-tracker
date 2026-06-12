package app

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type ThreadUseCase struct {
	threadRepo domain.ThreadRepo
}

func NewThreadUseCase(threadRepo domain.ThreadRepo) *ThreadUseCase {
	return &ThreadUseCase{threadRepo: threadRepo}
}

func (uc *ThreadUseCase) List(ctx context.Context, userID uuid.UUID, filter domain.ThreadFilter) ([]*domain.EmailThread, int, error) {
	if filter.Limit <= 0 {
		filter.Limit = 50
	}
	if filter.Page <= 0 {
		filter.Page = 1
	}
	return uc.threadRepo.ListByUser(ctx, userID, filter)
}

func (uc *ThreadUseCase) Update(ctx context.Context, userID, threadID uuid.UUID, status, notes string) error {
	thread, err := uc.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread.UserID != userID {
		return domain.ErrUnauthorized
	}

	if status != "" {
		thread.Status = status
	}
	if notes != "" {
		thread.Notes = notes
	}

	if err := uc.threadRepo.Update(ctx, thread); err != nil {
		return fmt.Errorf("update thread: %w", err)
	}
	return nil
}

func (uc *ThreadUseCase) RequestResummary(ctx context.Context, userID, threadID uuid.UUID) error {
	thread, err := uc.threadRepo.GetByID(ctx, threadID)
	if err != nil {
		return err
	}
	if thread.UserID != userID {
		return domain.ErrUnauthorized
	}
	return uc.threadRepo.UpdateSummaryStatus(ctx, threadID, domain.SummaryStatusPending)
}
