package authn

import "context"

type Principal struct {
	Subject    string
	Username   string
	Groups     []string
	IsAdmin    bool
	AuthMethod string
}

type contextKey string

const principalContextKey contextKey = "principal"

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	value := ctx.Value(principalContextKey)
	if value == nil {
		return Principal{}, false
	}
	principal, ok := value.(Principal)
	return principal, ok
}
