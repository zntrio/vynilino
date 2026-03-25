package app

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"
	"unicode"

	"github.com/google/uuid"
	pasetov4 "zntr.io/paseto/v4"

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
	users       domain.UserRepository
	tokens      domain.TokenRepository
	symmetricKey *pasetov4.LocalKey
	singleOwner bool
}

// NewUserService constructs a UserService.
func NewUserService(
	users domain.UserRepository,
	tokens domain.TokenRepository,
	tokenKeyHex string,
	singleOwner bool,
) (*UserService, error) {
	key, err := hexToLocalKey(tokenKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid token key: %w", err)
	}
	return &UserService{
		users:        users,
		tokens:       tokens,
		symmetricKey: key,
		singleOwner:  singleOwner,
	}, nil
}

// Register creates a new user account. In single-owner mode only the first
// account is allowed.
func (s *UserService) Register(ctx context.Context, email, password string) (*TokenPair, error) {
	if s.singleOwner {
		count, err := s.users.Count(ctx)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, domain.ErrRegistrationClosed
		}
	}

	if err := validatePassword(password); err != nil {
		return nil, err
	}

	hash, err := hashPassword(password)
	if err != nil {
		return nil, err
	}

	count, _ := s.users.Count(ctx)
	role := domain.RoleUser
	if count == 0 {
		role = domain.RoleAdmin
	}

	user, err := s.users.Create(ctx, &domain.User{
		Email:        email,
		PasswordHash: hash,
		Role:         role,
	})
	if err != nil {
		return nil, err
	}

	return s.issueTokenPair(ctx, user.ID)
}

// Login authenticates with email/password and returns a token pair.
func (s *UserService) Login(ctx context.Context, email, password string) (*TokenPair, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		// Constant-time response to prevent user enumeration.
		constantTimeReject()
		return nil, domain.ErrInvalidCredentials
	}

	if !user.Active {
		return nil, domain.ErrAccountDisabled
	}

	if user.IsLocked(time.Now()) {
		return nil, domain.ErrAccountLocked
	}

	if err := checkPassword(password, user.PasswordHash); err != nil {
		lockUntil := s.computeLockout(user)
		_ = s.users.RecordLoginFailure(ctx, user.ID, lockUntil)
		if lockUntil != nil {
			return nil, domain.ErrAccountLocked
		}
		return nil, domain.ErrInvalidCredentials
	}

	_ = s.users.ResetLoginFailure(ctx, user.ID)
	return s.issueTokenPair(ctx, user.ID)
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
func (s *UserService) ValidateAccessToken(token string) (string, error) {
	payload, err := pasetov4.Decrypt(s.symmetricKey, token, nil, nil)
	if err != nil {
		return "", domain.ErrInvalidToken
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
