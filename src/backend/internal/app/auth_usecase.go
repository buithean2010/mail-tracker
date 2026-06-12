package app

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"golang.org/x/oauth2"

	"github.com/buithean2010/mail-tracker/backend/internal/config"
	"github.com/buithean2010/mail-tracker/backend/internal/domain"
)

type AuthUseCase struct {
	cfg         *config.Config
	oauth2Cfg   *oauth2.Config
	userRepo    domain.UserRepo
	sessionRepo domain.SessionRepo
	tokenRepo   domain.TokenRepo
}

func NewAuthUseCase(cfg *config.Config, userRepo domain.UserRepo, sessionRepo domain.SessionRepo, tokenRepo domain.TokenRepo) *AuthUseCase {
	oauthCfg := &oauth2.Config{
		ClientID:     cfg.AzureClientID,
		ClientSecret: cfg.AzureClientSecret,
		RedirectURL:  cfg.AzureRedirectURI,
		Scopes:       []string{"openid", "profile", "email", "offline_access", "Mail.Read"},
		Endpoint: oauth2.Endpoint{
			AuthURL:  fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/authorize", cfg.AzureTenantID),
			TokenURL: fmt.Sprintf("https://login.microsoftonline.com/%s/oauth2/v2.0/token", cfg.AzureTenantID),
		},
	}
	return &AuthUseCase{cfg: cfg, oauth2Cfg: oauthCfg, userRepo: userRepo, sessionRepo: sessionRepo, tokenRepo: tokenRepo}
}

func (uc *AuthUseCase) BuildAuthURL(state, verifier string) string {
	return uc.oauth2Cfg.AuthCodeURL(state, oauth2.S256ChallengeOption(verifier))
}

func (uc *AuthUseCase) ExchangeCode(ctx context.Context, code, verifier string) (*domain.UserToken, *MSUserInfo, error) {
	token, err := uc.oauth2Cfg.Exchange(ctx, code, oauth2.VerifierOption(verifier))
	if err != nil {
		return nil, nil, fmt.Errorf("oauth exchange: %w", err)
	}

	userInfo, err := fetchMSUserInfo(ctx, token.AccessToken)
	if err != nil {
		return nil, nil, fmt.Errorf("fetch user info: %w", err)
	}

	ut := &domain.UserToken{
		AccessToken:  token.AccessToken,
		RefreshToken: token.RefreshToken,
		ExpiresAt:    token.Expiry,
	}

	return ut, userInfo, nil
}

func (uc *AuthUseCase) UpsertUserAndToken(ctx context.Context, info *MSUserInfo, ut *domain.UserToken) (*domain.User, error) {
	user := &domain.User{
		MicrosoftID: info.ID,
		Email:       info.Mail,
		DisplayName: info.DisplayName,
		FilterSettings: json.RawMessage("{}"),
	}

	if err := uc.userRepo.Upsert(ctx, user); err != nil {
		return nil, fmt.Errorf("upsert user: %w", err)
	}

	saved, err := uc.userRepo.GetByMicrosoftID(ctx, info.ID)
	if err != nil {
		return nil, err
	}

	ut.UserID = saved.ID
	if err := uc.tokenRepo.Upsert(ctx, ut); err != nil {
		return nil, fmt.Errorf("upsert token: %w", err)
	}

	// Clear needs_reauth if it was set
	if saved.NeedsReauth {
		_ = uc.userRepo.SetNeedsReauth(ctx, saved.ID, false)
	}

	return saved, nil
}

func (uc *AuthUseCase) CreateSession(ctx context.Context, user *domain.User) (*domain.Session, error) {
	id, err := randomHex(32)
	if err != nil {
		return nil, err
	}

	session := &domain.Session{
		ID:        id,
		UserID:    user.ID,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	if err := uc.sessionRepo.Create(ctx, session); err != nil {
		return nil, fmt.Errorf("create session: %w", err)
	}

	return session, nil
}

func (uc *AuthUseCase) DeleteSession(ctx context.Context, sessionID string) error {
	return uc.sessionRepo.Delete(ctx, sessionID)
}

func (uc *AuthUseCase) RefreshToken(ctx context.Context, _ interface{}, rt string) (*domain.UserToken, error) {
	ts := uc.oauth2Cfg.TokenSource(ctx, &oauth2.Token{RefreshToken: rt})
	newToken, err := ts.Token()
	if err != nil {
		return nil, fmt.Errorf("token refresh: %w", err)
	}

	return &domain.UserToken{
		AccessToken:  newToken.AccessToken,
		RefreshToken: newToken.RefreshToken,
		ExpiresAt:    newToken.Expiry,
	}, nil
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// MSUserInfo is the minimal response from GET /me
type MSUserInfo struct {
	ID          string `json:"id"`
	Mail        string `json:"mail"`
	DisplayName string `json:"displayName"`
}
