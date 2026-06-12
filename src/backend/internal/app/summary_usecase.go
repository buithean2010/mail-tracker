package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

// Concurrency limits: max parallel users and per-provider AI request limits.
const (
	maxConcurrentUsers     = 8
	maxOpenAIConcurrent    = 5
	maxOpenRouterConcurrent = 5
	maxPAConcurrent        = 3
)

type SummaryUseCase struct {
	threadRepo    domain.ThreadRepo
	apiKeyRepo    domain.APIKeyRepo
	openaiCli     domain.AIClient
	openrouterCli domain.AIClient
	paCli         domain.PAClient
	enc           domain.Encryptor

	semOpenAI     chan struct{}
	semOpenRouter chan struct{}
	semPA         chan struct{}
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
		semOpenAI:     make(chan struct{}, maxOpenAIConcurrent),
		semOpenRouter: make(chan struct{}, maxOpenRouterConcurrent),
		semPA:         make(chan struct{}, maxPAConcurrent),
	}
}

// RunAll processes all pending summary threads in parallel, grouped by user.
func (uc *SummaryUseCase) RunAll(ctx context.Context) error {
	threads, err := uc.threadRepo.GetPendingSummaries(ctx)
	if err != nil {
		return fmt.Errorf("get pending summaries: %w", err)
	}

	byUser := make(map[uuid.UUID][]*domain.EmailThread)
	for _, t := range threads {
		byUser[t.UserID] = append(byUser[t.UserID], t)
	}

	userSem := make(chan struct{}, maxConcurrentUsers)
	var wg sync.WaitGroup

	for userID, userThreads := range byUser {
		wg.Add(1)
		userSem <- struct{}{}
		go func(uid uuid.UUID, ts []*domain.EmailThread) {
			defer wg.Done()
			defer func() { <-userSem }()
			log.Printf("[job2] user %s: summarizing %d threads", uid, len(ts))
			for _, t := range ts {
				if err := uc.summarize(ctx, t); err != nil {
					log.Printf("[job2] thread %s: %v", t.ID, err)
				}
			}
		}(userID, userThreads)
	}

	wg.Wait()
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
		var sem chan struct{}
		switch key.Provider {
		case domain.AIProviderOpenRouter:
			cli = uc.openrouterCli
			sem = uc.semOpenRouter
		default:
			cli = uc.openaiCli
			sem = uc.semOpenAI
		}
		sem <- struct{}{}
		result, err = cli.Summarize(ctx, key.APIKey, key.Model, messages)
		<-sem

	case domain.AIModePA:
		uc.semPA <- struct{}{}
		result, err = uc.paCli.Summarize(ctx, key.PAWebhookURL, messages)
		<-uc.semPA

	default:
		return uc.threadRepo.UpdateSummaryStatus(ctx, thread.ID, domain.SummaryStatusNoKey)
	}

	if err != nil {
		log.Printf("AI call failed for thread %s: %v", thread.ID, err)
		return uc.threadRepo.UpdateSummaryStatus(ctx, thread.ID, domain.SummaryStatusFailed)
	}

	return uc.threadRepo.UpdateSummary(ctx, thread.ID, result)
}
