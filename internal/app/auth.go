package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"
	"unicode"

	"github.com/google/uuid"

	pasetov4 "zntr.io/paseto/v4"

	"zntr.io/vynilino/internal/ctxutil"
	"zntr.io/vynilino/internal/domain"
)

const (
	accessTokenTTL  = 15 * time.Minute
	refreshTokenTTL = 30 * 24 * time.Hour

	maxFailedLogins = 10
	lockoutDuration = 15 * time.Minute
)

// TokenPair holds an access/refresh token pair.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int // access token TTL in seconds
}

// accessClaims is the payload embedded in a PASETO v4 local token.
type accessClaims struct {
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	NotBefore int64  `json:"nbf"`
	ExpiresAt int64  `json:"exp"`
}

// UserService handles user registration, authentication, and token lifecycle.
type UserService struct {
	users        domain.UserRepository
	tokens       domain.TokenRepository
	symmetricKey *pasetov4.LocalKey
	// secondaryKey is the previous key retained during a rotation bridge period.
	// When non-nil, ValidateAccessToken will fall back to it if decryption with
	// symmetricKey fails, keeping sessions alive for one access-token TTL.
	secondaryKey   *pasetov4.LocalKey
	singleOwner    bool
	bootstrapToken string     // THREAT-001: required token for first-user registration
	bootstrapMu    sync.Mutex // THREAT-002: serialises Count+Create for first-user provisioning
}

// NewUserService constructs a UserService.
// tokenKeyNewHex is optional: pass a non-empty hex string during key rotation to
// retain the old key as a secondary decryption fallback for the bridge period.
// bootstrapToken is optional: when non-empty, first-user registration requires a
// matching X-Bootstrap-Token header (THREAT-001 mitigation).
func NewUserService(
	users domain.UserRepository,
	tokens domain.TokenRepository,
	tokenKeyHex string,
	tokenKeyNewHex string,
	singleOwner bool,
	bootstrapToken string,
) (*UserService, error) {
	key, err := hexToLocalKey(tokenKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid token key: %w", err)
	}
	svc := &UserService{
		users:          users,
		tokens:         tokens,
		symmetricKey:   key,
		singleOwner:    singleOwner,
		bootstrapToken: bootstrapToken,
	}
	if tokenKeyNewHex != "" {
		secondary, err := hexToLocalKey(tokenKeyNewHex)
		if err != nil {
			return nil, fmt.Errorf("invalid secondary token key: %w", err)
		}
		svc.secondaryKey = secondary
	}
	return svc, nil
}

// Register creates a new user account. In single-owner mode only the first
// account is allowed.
func (s *UserService) Register(ctx context.Context, email, password string) (*TokenPair, error) {
	// THREAT-001: when bootstrapToken is configured, the first registration
	// requires a matching X-Bootstrap-Token header value in context.
	if s.bootstrapToken != "" {
		count, err := s.users.Count(ctx)
		if err != nil {
			return nil, err
		}
		if count == 0 {
			if provided := ctxutil.BootstrapTokenFromContext(ctx); provided != s.bootstrapToken {
				return nil, domain.ErrBootstrapTokenRequired
			}
		}
	}

	if err := validatePassword(password); err != nil {
		return nil, err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	// THREAT-002: atomic Count+Create under mutex via bootstrapCreate.
	user, err := s.bootstrapCreate(ctx, email, hash)
	if err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, user.ID)
}

// bootstrapCreate atomically checks whether this is the first user and creates
// the account with the appropriate role. It must be the only code path that
// creates users, covering both local registration and OIDC provisioning.
func (s *UserService) bootstrapCreate(ctx context.Context, email, passwordHash string) (*domain.User, error) {
	s.bootstrapMu.Lock()
	defer s.bootstrapMu.Unlock()

	count, err := s.users.Count(ctx)
	if err != nil {
		return nil, err
	}
	if s.singleOwner && count > 0 {
		return nil, domain.ErrRegistrationClosed
	}

	role := domain.RoleUser
	if count == 0 {
		role = domain.RoleAdmin
	}

	return s.users.Create(ctx, &domain.User{
		Email:        email,
		PasswordHash: passwordHash,
		Role:         role,
	})
}

// BootstrapProvisionUser creates a new OIDC-provisioned user via the same
// atomic bootstrap path as Register. Email may be empty when the provider did
// not supply a verified email address.
func (s *UserService) BootstrapProvisionUser(ctx context.Context, email string) (*domain.User, error) {
	return s.bootstrapCreate(ctx, email, "")
}

// Login authenticates with email/password and returns a token pair.
func (s *UserService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Constant-time response to prevent user enumeration.
		constantTimeReject()
		slog.InfoContext(ctx, "auth.login", "outcome", "invalid_credentials")
		return nil, domain.ErrInvalidCredentials
	}

	if !user.Active {
		slog.InfoContext(ctx, "auth.login", "outcome", "invalid_credentials")
		return nil, domain.ErrAccountDisabled
	}

	if user.IsLocked(time.Now()) {
		slog.InfoContext(ctx, "auth.login", "outcome", "account_locked")
		return nil, domain.ErrAccountLocked
	}

	if err := checkPassword(password, user.PasswordHash); err != nil {
		lockUntil := s.computeLockout(user)
		_ = s.users.RecordLoginFailure(ctx, user.ID, lockUntil)
		if lockUntil != nil {
			slog.InfoContext(ctx, "auth.login", "outcome", "account_locked")
			return nil, domain.ErrAccountLocked
		}
		slog.InfoContext(ctx, "auth.login", "outcome", "invalid_credentials")
		return nil, domain.ErrInvalidCredentials
	}

	_ = s.users.ResetLoginFailure(ctx, user.ID)
	pair, err := s.issueTokenPair(ctx, user.ID)
	if err != nil {
		slog.InfoContext(ctx, "auth.login", "outcome", "server_error")
		return nil, err
	}
	slog.InfoContext(ctx, "auth.login", "outcome", "success", "user_id", user.ID)
	return pair, nil
}

// RefreshToken issues a new access token, rotating the refresh token.
func (s *UserService) RefreshToken(ctx context.Context, rawRefreshToken string) (*TokenPair, error) {
	hash := hashToken(rawRefreshToken)
	stored, err := s.tokens.GetByHash(ctx, hash)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if stored.Revoked || time.Now().After(stored.ExpiresAt) {
		return nil, domain.ErrInvalidToken
	}

	user, err := s.users.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, domain.ErrInvalidToken
	}
	if !user.Active {
		return nil, domain.ErrAccountDisabled
	}

	// Rotate: revoke old token before issuing new pair.
	if err := s.tokens.Revoke(ctx, stored.ID); err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, stored.UserID)
}

// Logout revokes all refresh tokens for the current user.
func (s *UserService) Logout(ctx context.Context, userID string) error {
	return s.tokens.RevokeAllForUser(ctx, userID)
}

// ValidateAccessToken parses and validates a PASETO v4 local token, returning the user ID.
// During key rotation, if decryption with the primary key fails and a secondary key is
// configured, it retries with the secondary key to keep sessions alive through the bridge period.
func (s *UserService) ValidateAccessToken(token string) (string, error) {
	payload, err := pasetov4.Decrypt(s.symmetricKey, token, nil, nil)
	if err != nil {
		if s.secondaryKey == nil {
			return "", domain.ErrInvalidToken
		}
		payload, err = pasetov4.Decrypt(s.secondaryKey, token, nil, nil)
		if err != nil {
			return "", domain.ErrInvalidToken
		}
	}

	var claims accessClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return "", domain.ErrInvalidToken
	}

	now := time.Now().Unix()
	if now < claims.NotBefore || now > claims.ExpiresAt {
		return "", domain.ErrInvalidToken
	}
	if claims.Subject == "" {
		return "", domain.ErrInvalidToken
	}

	return claims.Subject, nil
}

// --- internals ---

func (s *UserService) issueTokenPair(ctx context.Context, userID string) (*TokenPair, error) {
	accessToken, err := s.mintAccessToken(userID)
	if err != nil {
		return nil, err
	}

	rawRefresh := uuid.NewString()
	expiresAt := time.Now().Add(refreshTokenTTL)
	_, err = s.tokens.Create(ctx, &domain.RefreshToken{
		UserID:    userID,
		TokenHash: hashToken(rawRefresh),
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: rawRefresh,
		ExpiresIn:    int(accessTokenTTL.Seconds()),
	}, nil
}

func (s *UserService) mintAccessToken(userID string) (string, error) {
	now := time.Now()
	claims := accessClaims{
		Subject:   userID,
		IssuedAt:  now.Unix(),
		NotBefore: now.Unix(),
		ExpiresAt: now.Add(accessTokenTTL).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	return pasetov4.Encrypt(rand.Reader, s.symmetricKey, payload, nil, nil)
}

func (s *UserService) computeLockout(user *domain.User) *time.Time {
	if user.FailedLoginCount+1 >= maxFailedLogins {
		t := time.Now().Add(lockoutDuration)
		return &t
	}
	return nil
}

// hashToken returns a SHA-256 hex digest of the raw token value.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func hexToLocalKey(hexStr string) (*pasetov4.LocalKey, error) {
	b, err := hex.DecodeString(hexStr)
	if err != nil {
		return nil, fmt.Errorf("decode hex: %w", err)
	}
	return pasetov4.LocalKeyFromSeed(b)
}

func validatePassword(pw string) error {
	if len(pw) < 12 {
		return domain.ErrWeakPassword
	}
	var hasUpper, hasLower, hasDigit bool
	for _, r := range pw {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasUpper || !hasLower || !hasDigit {
		return fmt.Errorf("%w: must contain uppercase, lowercase, and a digit", domain.ErrWeakPassword)
	}
	return nil
}
