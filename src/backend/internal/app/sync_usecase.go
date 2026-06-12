package app

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type SyncUseCase struct {
	userRepo    domain.UserRepo
	sessionRepo domain.SessionRepo
	tokenRepo   domain.TokenRepo
	threadRepo  domain.ThreadRepo
	mailClient  domain.MailClient
	broker      domain.SSEBroker
	enc         domain.Encryptor
	authUC      *AuthUseCase
}

func NewSyncUseCase(
	userRepo domain.UserRepo,
	sessionRepo domain.SessionRepo,
	tokenRepo domain.TokenRepo,
	threadRepo domain.ThreadRepo,
	mailClient domain.MailClient,
	broker domain.SSEBroker,
	enc domain.Encryptor,
	authUC *AuthUseCase,
) *SyncUseCase {
	return &SyncUseCase{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
		tokenRepo:   tokenRepo,
		threadRepo:  threadRepo,
		mailClient:  mailClient,
		broker:      broker,
		enc:         enc,
		authUC:      authUC,
	}
}

func (uc *SyncUseCase) SyncAll(ctx context.Context) error {
	users, err := uc.sessionRepo.GetActiveUsers(ctx, 7)
	if err != nil {
		return fmt.Errorf("get active users: %w", err)
	}

	for _, user := range users {
		if err := uc.syncUser(ctx, user); err != nil {
			log.Printf("sync user %s: %v", user.ID, err)
		}
	}
	return nil
}

func (uc *SyncUseCase) SyncUser(ctx context.Context, userID uuid.UUID) error {
	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}
	return uc.syncUser(ctx, user)
}

func (uc *SyncUseCase) syncUser(ctx context.Context, user *domain.User) error {
	token, err := uc.tokenRepo.GetByUserID(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("get token: %w", err)
	}

	if token.NeedsRefresh() {
		newToken, err := uc.authUC.RefreshToken(ctx, user.ID, token.RefreshToken)
		if err != nil {
			_ = uc.userRepo.SetNeedsReauth(ctx, user.ID, true)
			return fmt.Errorf("refresh token: %w", err)
		}
		newToken.UserID = user.ID
		if err := uc.tokenRepo.Upsert(ctx, newToken); err != nil {
			return fmt.Errorf("save refreshed token: %w", err)
		}
		token = newToken
	}

	filter, err := domain.ParseFilterSettings(user.FilterSettings)
	if err != nil {
		log.Printf("user %s: invalid filter settings, syncing all", user.ID)
		filter = &domain.FilterSettings{}
	}

	since := time.Now().Add(-30 * 24 * time.Hour) // default: 30 days on first sync
	if user.LastSyncedAt != nil {
		since = *user.LastSyncedAt
	}
	syncStarted := time.Now().UTC()

	messages, err := uc.mailClient.FetchMessages(ctx, token.AccessToken, since, filter.WatchedFolders)
	if err != nil {
		return fmt.Errorf("fetch messages: %w", err)
	}

	grouped := groupByConversation(messages)

	for convID, msgs := range grouped {
		// Apply filter: keep conversation if any message matches
		matched := false
		for _, msg := range msgs {
			if filter.MyEmail == "" || filter.Matches(msg) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}

		sort.Slice(msgs, func(i, j int) bool {
			return msgs[i].ReceivedAt.Before(msgs[j].ReceivedAt)
		})

		rawJSON, _ := json.Marshal(msgs)
		thread := &domain.EmailThread{
			UserID:         user.ID,
			ConversationID: convID,
			Subject:        msgs[len(msgs)-1].Subject,
			FromAddress:    msgs[0].From,
			ReceivedAt:     msgs[0].ReceivedAt,
			LastMessageAt:  msgs[len(msgs)-1].ReceivedAt,
			MessageCount:   len(msgs),
			RawMessages:    rawJSON,
			SummaryStatus:  domain.SummaryStatusPending,
		}

		if err := uc.threadRepo.Upsert(ctx, thread); err != nil {
			log.Printf("upsert thread %s: %v", convID, err)
		}
	}

	if err := uc.userRepo.SetLastSyncedAt(ctx, user.ID, syncStarted); err != nil {
		log.Printf("set last_synced_at user %s: %v", user.ID, err)
	}

	uc.broker.Publish(user.ID, "sync_complete", map[string]string{"user_id": user.ID.String()})
	return nil
}

func groupByConversation(messages []*domain.MailMessage) map[string][]*domain.MailMessage {
	groups := make(map[string][]*domain.MailMessage)
	for _, m := range messages {
		groups[m.ConversationID] = append(groups[m.ConversationID], m)
	}
	return groups
}
