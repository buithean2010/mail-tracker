package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type SummaryUseCase struct {
	threadRepo  domain.ThreadRepo
	apiKeyRepo  domain.APIKeyRepo
	openaiCli   domain.AIClient
	openrouterCli domain.AIClient
	paCli       domain.PAClient
	enc         domain.Encryptor
}

func NewSummaryUseCase(
	threadRepo domain.ThreadRepo,
	apiKeyRepo domain.APIKeyRepo,
	openaiCli domain.AIClient,
	openrouterCli domain.AIClient,
	paCli domain.PAClient,
	enc domain.Encryptor,
) *SummaryUseCase {
	return &SummaryUseCase{
		threadRepo:    threadRepo,
		apiKeyRepo:    apiKeyRepo,
		openaiCli:     openaiCli,
		openrouterCli: openrouterCli,
		paCli:         paCli,
		enc:           enc,
	}
}

func (uc *SummaryUseCase) RunAll(ctx context.Context) error {
	threads, err := uc.threadRepo.GetPendingSummaries(ctx)
	if err != nil {
		return fmt.Errorf("get pending summaries: %w", err)
	}

	for _, t := range threads {
		if err := uc.summarize(ctx, t); err != nil {
			log.Printf("summarize thread %s: %v", t.ID, err)
		}
	}
	return nil
}

func (uc *SummaryUseCase) SummarizeThread(ctx context.Context, threadID string) error {
	// placeholder for on-demand re-summary via API
	return nil
}

func (uc *SummaryUseCase) summarize(ctx context.Context, thread *domain.EmailThread) error {
	if err := uc.threadRepo.UpdateSummaryStatus(ctx, thread.ID, domain.SummaryStatusProcessing); err != nil {
		return err
	}

	key, err := uc.apiKeyRepo.GetByUserID(ctx, thread.UserID)
	if err != nil {
		// No key configured → Tier 1
		return uc.threadRepo.UpdateSummaryStatus(ctx, thread.ID, domain.SummaryStatusNoKey)
	}

	var messages []*domain.MailMessage
	if err := json.Unmarshal(thread.RawMessages, &messages); err != nil {
		return uc.threadRepo.UpdateSummaryStatus(ctx, thread.ID, domain.SummaryStatusFailed)
	}

	var result *domain.SummaryResult

	switch key.AIMode {
	case domain.AIModeByok:
		var cli domain.AIClient
		switch key.Provider {
		case domain.AIProviderOpenRouter:
			cli = uc.openrouterCli
		default:
			cli = uc.openaiCli
		}
		result, err = cli.Summarize(ctx, key.APIKey, key.Model, messages)

	case domain.AIModePA:
		result, err = uc.paCli.Summarize(ctx, key.PAWebhookURL, messages)

	default:
		return uc.threadRepo.UpdateSummaryStatus(ctx, thread.ID, domain.SummaryStatusNoKey)
	}

	if err != nil {
		log.Printf("AI call failed for thread %s: %v", thread.ID, err)
		return uc.threadRepo.UpdateSummaryStatus(ctx, thread.ID, domain.SummaryStatusFailed)
	}

	return uc.threadRepo.UpdateSummary(ctx, thread.ID, result)
}
