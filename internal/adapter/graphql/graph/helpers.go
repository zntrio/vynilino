package graph

import (
	"errors"
	"fmt"

	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/domain"
)

var errUnauthenticated = fmt.Errorf("UNAUTHENTICATED")

func gqlErr(err error) error {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		return fmt.Errorf("INVALID_CREDENTIALS")
	case errors.Is(err, domain.ErrAccountLocked):
		return fmt.Errorf("ACCOUNT_LOCKED")
	case errors.Is(err, domain.ErrRegistrationClosed):
		return fmt.Errorf("REGISTRATION_CLOSED")
	case errors.Is(err, domain.ErrWeakPassword):
		return fmt.Errorf("WEAK_PASSWORD: %w", err)
	case errors.Is(err, domain.ErrEmailTaken):
		return fmt.Errorf("EMAIL_TAKEN")
	case errors.Is(err, domain.ErrOIDCNotConfigured):
		return fmt.Errorf("NOT_CONFIGURED")
	case errors.Is(err, domain.ErrOIDCInvalidState):
		return fmt.Errorf("OIDC_INVALID_STATE")
	case errors.Is(err, domain.ErrOIDCStateExpired):
		return fmt.Errorf("OIDC_STATE_EXPIRED")
	case errors.Is(err, domain.ErrOIDCTokenInvalid):
		return fmt.Errorf("OIDC_TOKEN_INVALID")
	case errors.Is(err, domain.ErrOIDCUserForbidden):
		return fmt.Errorf("FORBIDDEN")
	default:
		return err
	}
}

func authPayload(pair *app.TokenPair, user *domain.User) *AuthPayload {
	return &AuthPayload{
		AccessToken:  pair.AccessToken,
		RefreshToken: pair.RefreshToken,
		ExpiresIn:    pair.ExpiresIn,
		User:         user,
	}
}
