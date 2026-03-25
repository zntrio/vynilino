package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"

	"zntr.io/vynilino/internal/config"
	"zntr.io/vynilino/internal/domain"
)

const oidcStateTTL = 5 * time.Minute

// OIDCService handles the OIDC Authorization Code + PKCE flow.
type OIDCService struct {
	cfg          *config.Config
	userSvc      *UserService
	identityRepo domain.OIDCIdentityRepository
	stateRepo    domain.OIDCStateRepository

	once     sync.Once
	provider *gooidc.Provider
	provErr  error
}

// NewOIDCService constructs an OIDCService. Returns (nil, nil) when OIDC is
// not configured, allowing fx to treat it as an optional dependency.
func NewOIDCService(
	cfg *config.Config,
	userSvc *UserService,
	identityRepo domain.OIDCIdentityRepository,
	stateRepo domain.OIDCStateRepository,
) (*OIDCService, error) {
	if cfg.OIDCIssuer == "" {
		return nil, nil
	}
	return &OIDCService{
		cfg:          cfg,
		userSvc:      userSvc,
		identityRepo: identityRepo,
		stateRepo:    stateRepo,
	}, nil
}

// AuthorizationURL generates an OIDC authorization URL with PKCE (S256).
// It persists the state/nonce/verifier server-side and returns the redirect URL.
func (s *OIDCService) AuthorizationURL(ctx context.Context) (url, state string, err error) {
	provider, err := s.getProvider(ctx)
	if err != nil {
		return "", "", fmt.Errorf("oidc provider: %w", err)
	}

	stateVal, err := randomHex(32)
	if err != nil {
		return "", "", err
	}
	nonce, err := randomHex(32)
	if err != nil {
		return "", "", err
	}
	codeVerifier, err := randomCodeVerifier()
	if err != nil {
		return "", "", err
	}
	codeChallenge := pkceS256(codeVerifier)

	// Lazy GC: prune expired states before creating a new one.
	_ = s.stateRepo.DeleteExpired(ctx, time.Now().Add(-oidcStateTTL))

	if err := s.stateRepo.Create(ctx, &domain.OIDCState{
		State:        stateVal,
		Nonce:        nonce,
		CodeVerifier: codeVerifier,
	}); err != nil {
		return "", "", err
	}

	oauth2Cfg := s.oauth2Config(provider)
	authURL := oauth2Cfg.AuthCodeURL(
		stateVal,
		oauth2.SetAuthURLParam("nonce", nonce),
		oauth2.SetAuthURLParam("code_challenge", codeChallenge),
		oauth2.SetAuthURLParam("code_challenge_method", "S256"),
	)
	return authURL, stateVal, nil
}

// HandleCallback processes the OIDC callback: validates state, exchanges code,
// verifies ID token, resolves or provisions the user, and issues vynilino tokens.
func (s *OIDCService) HandleCallback(ctx context.Context, code, stateVal string) (*TokenPair, error) {
	storedState, err := s.stateRepo.FindByState(ctx, stateVal)
	if err != nil {
		if err == domain.ErrNotFound {
			return nil, domain.ErrOIDCInvalidState
		}
		return nil, err
	}

	// Delete state immediately to prevent replay.
	_ = s.stateRepo.Delete(ctx, stateVal)

	if time.Since(storedState.CreatedAt) > oidcStateTTL {
		return nil, domain.ErrOIDCStateExpired
	}

	provider, err := s.getProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("oidc provider: %w", err)
	}

	oauth2Cfg := s.oauth2Config(provider)
	oauth2Token, err := oauth2Cfg.Exchange(
		ctx, code,
		oauth2.SetAuthURLParam("code_verifier", storedState.CodeVerifier),
	)
	if err != nil {
		return nil, domain.ErrOIDCTokenInvalid
	}

	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		return nil, domain.ErrOIDCTokenInvalid
	}

	verifier := provider.Verifier(&gooidc.Config{ClientID: s.cfg.OIDCClientID})
	idToken, err := verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, domain.ErrOIDCTokenInvalid
	}

	if idToken.Nonce != storedState.Nonce {
		return nil, domain.ErrOIDCTokenInvalid
	}

	var claims struct {
		Subject       string `json:"sub"`
		Email         string `json:"email"`
		EmailVerified bool   `json:"email_verified"`
	}
	if err := idToken.Claims(&claims); err != nil {
		return nil, domain.ErrOIDCTokenInvalid
	}

	userID, err := s.resolveUser(ctx, s.cfg.OIDCIssuer, claims.Subject, claims.Email, claims.EmailVerified)
	if err != nil {
		return nil, err
	}

	user, err := s.userSvc.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !user.Active {
		return nil, domain.ErrAccountDisabled
	}

	return s.userSvc.issueTokenPair(ctx, userID)
}

// resolveUser finds or provisions a vynilino user for the given OIDC identity.
func (s *OIDCService) resolveUser(ctx context.Context, provider, subject, email string, emailVerified bool) (string, error) {
	// 1. Check existing OIDC identity link.
	identity, err := s.identityRepo.FindByProviderSubject(ctx, provider, subject)
	if err == nil {
		return identity.UserID, nil
	}
	if err != domain.ErrNotFound {
		return "", err
	}

	// 2. Try to link by verified email.
	if emailVerified && email != "" {
		user, err := s.userSvc.users.GetByEmail(ctx, email)
		if err == nil {
			if linkErr := s.identityRepo.Create(ctx, &domain.OIDCIdentity{
				UserID:   user.ID,
				Provider: provider,
				Subject:  subject,
			}); linkErr != nil {
				return "", linkErr
			}
			return user.ID, nil
		}
	}

	// 3. Auto-provision a new account via the shared atomic bootstrap path.
	// Only assign email if it's verified; unverified emails are not trustworthy
	// and may collide with an existing local account.
	provisionEmail := ""
	if emailVerified {
		provisionEmail = email
	}

	newUser, err := s.userSvc.BootstrapProvisionUser(ctx, provisionEmail)
	if err != nil {
		return "", err
	}

	if err := s.identityRepo.Create(ctx, &domain.OIDCIdentity{
		UserID:   newUser.ID,
		Provider: provider,
		Subject:  subject,
	}); err != nil {
		return "", err
	}

	return newUser.ID, nil
}

func (s *OIDCService) getProvider(ctx context.Context) (*gooidc.Provider, error) {
	s.once.Do(func() {
		s.provider, s.provErr = gooidc.NewProvider(ctx, s.cfg.OIDCIssuer)
	})
	return s.provider, s.provErr
}

func (s *OIDCService) oauth2Config(provider *gooidc.Provider) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.cfg.OIDCClientID,
		ClientSecret: s.cfg.OIDCClientSecret,
		RedirectURL:  s.cfg.OIDCRedirectURL,
		Endpoint:     provider.Endpoint(),
		Scopes:       []string{gooidc.ScopeOpenID, "email", "profile"},
	}
}

// --- helpers ---

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// randomCodeVerifier generates a PKCE code verifier (43–128 base64url chars).
func randomCodeVerifier() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// pkceS256 computes BASE64URL(SHA256(verifier)).
func pkceS256(verifier string) string {
	h := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(h[:])
}
