package graph

import (
	"zntr.io/vynilino/internal/app"
	"zntr.io/vynilino/internal/domain"
)

// Resolver holds application services injected by the DI container.
type Resolver struct {
	UserSvc    *app.UserService
	RecordSvc  *app.RecordService
	DiscogsSvc *app.DiscogsService
	UserRepo   domain.UserRepository
	OIDCSvc    *app.OIDCService // nil when OIDC is not configured
}
