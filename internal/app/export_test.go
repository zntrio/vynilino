package app

import "context"

// ResolveUserForTest exposes resolveUser for unit testing without a real OIDC provider.
func (s *OIDCService) ResolveUserForTest(ctx context.Context, provider, subject, email string, emailVerified bool) error {
	_, err := s.resolveUser(ctx, provider, subject, email, emailVerified)
	return err
}
