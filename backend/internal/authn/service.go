package authn

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strings"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/golang-jwt/jwt/v5"

	"github.com/runbook-hunter/runbook-hunter/backend/internal/config"
)

type Service struct {
	cfg          config.AuthConfig
	oidcVerifier *oidc.IDTokenVerifier
}

func NewService(ctx context.Context, cfg config.AuthConfig) (*Service, error) {
	service := &Service{cfg: cfg}
	mode := strings.ToLower(strings.TrimSpace(cfg.Mode))
	if mode == "oidc" {
		provider, err := oidc.NewProvider(ctx, strings.TrimSpace(cfg.OIDC.IssuerURL))
		if err != nil {
			return nil, fmt.Errorf("oidc provider init failed: %w", err)
		}
		service.oidcVerifier = provider.Verifier(&oidc.Config{
			ClientID: strings.TrimSpace(cfg.OIDC.ClientID),
		})
	}
	return service, nil
}

func (s *Service) AuthenticateRequest(r *http.Request) (Principal, error) {
	mode := strings.ToLower(strings.TrimSpace(s.cfg.Mode))
	switch mode {
	case "jwt":
		return s.authenticateJWT(r)
	case "oidc":
		return s.authenticateOIDC(r)
	default:
		return s.authenticateBasic(r)
	}
}

func (s *Service) authenticateBasic(r *http.Request) (Principal, error) {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return Principal{}, errors.New("basic credentials required")
	}
	if user != s.cfg.BasicUser || pass != s.cfg.BasicPass {
		return Principal{}, errors.New("invalid basic credentials")
	}
	return Principal{
		Subject:    user,
		Username:   user,
		IsAdmin:    true,
		AuthMethod: "basic",
	}, nil
}

func (s *Service) authenticateJWT(r *http.Request) (Principal, error) {
	tokenString, err := bearerToken(r)
	if err != nil {
		return Principal{}, err
	}
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.cfg.JWTSecret), nil
	})
	if err != nil || !token.Valid {
		return Principal{}, errors.New("invalid jwt token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return Principal{}, errors.New("invalid jwt claims")
	}
	principal := principalFromClaims(claims, s.cfg.OIDC.GroupsClaim, s.cfg.OIDC.UsernameClaim)
	principal.AuthMethod = "jwt"
	principal.IsAdmin = isAdminPrincipal(principal, s.cfg)
	return principal, nil
}

func (s *Service) authenticateOIDC(r *http.Request) (Principal, error) {
	tokenString, err := bearerToken(r)
	if err != nil {
		if s.cfg.OIDC.AllowBasicFallback {
			return s.authenticateBasic(r)
		}
		return Principal{}, err
	}
	if s.oidcVerifier == nil {
		return Principal{}, errors.New("oidc verifier is not configured")
	}
	idToken, err := s.oidcVerifier.Verify(r.Context(), tokenString)
	if err != nil {
		if s.cfg.OIDC.AllowBasicFallback {
			if principal, basicErr := s.authenticateBasic(r); basicErr == nil {
				return principal, nil
			}
		}
		return Principal{}, errors.New("invalid oidc token")
	}
	claims := map[string]any{}
	if err := idToken.Claims(&claims); err != nil {
		return Principal{}, errors.New("invalid oidc claims")
	}
	principal := principalFromClaims(claims, s.cfg.OIDC.GroupsClaim, s.cfg.OIDC.UsernameClaim)
	principal.AuthMethod = "oidc"
	principal.IsAdmin = isAdminPrincipal(principal, s.cfg)
	return principal, nil
}

func principalFromClaims(claims map[string]any, groupsClaim, usernameClaim string) Principal {
	subject := readClaimString(claims, "sub")
	username := readClaimString(claims, usernameClaim)
	if username == "" {
		username = readClaimString(claims, "preferred_username")
	}
	if username == "" {
		username = readClaimString(claims, "email")
	}
	groups := readClaimStringSlice(claims, groupsClaim)
	if len(groups) == 0 {
		groups = readClaimStringSlice(claims, "groups")
	}
	if len(groups) == 0 {
		groups = readRealmRoles(claims)
	}
	return Principal{
		Subject:  subject,
		Username: username,
		Groups:   normalizeGroups(groups),
	}
}

func isAdminPrincipal(principal Principal, cfg config.AuthConfig) bool {
	if strings.TrimSpace(principal.Username) != "" && principal.Username == cfg.BasicUser {
		return true
	}
	adminGroups := normalizeGroups(cfg.OIDC.AdminGroups)
	for _, group := range principal.Groups {
		if slices.Contains(adminGroups, group) {
			return true
		}
	}
	return false
}

func bearerToken(r *http.Request) (string, error) {
	authorization := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(authorization, "Bearer ") {
		return "", errors.New("missing bearer token")
	}
	tokenString := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
	if tokenString == "" {
		return "", errors.New("missing bearer token")
	}
	return tokenString, nil
}

func readClaimString(claims map[string]any, key string) string {
	value := strings.TrimSpace(key)
	if value == "" {
		return ""
	}
	raw, ok := claims[value]
	if !ok || raw == nil {
		return ""
	}
	switch item := raw.(type) {
	case string:
		return strings.TrimSpace(item)
	default:
		return ""
	}
}

func readClaimStringSlice(claims map[string]any, key string) []string {
	value := strings.TrimSpace(key)
	if value == "" {
		return nil
	}
	raw, ok := claims[value]
	if !ok || raw == nil {
		return nil
	}
	switch item := raw.(type) {
	case []any:
		out := make([]string, 0, len(item))
		for _, nested := range item {
			text, ok := nested.(string)
			if !ok {
				continue
			}
			text = strings.TrimSpace(text)
			if text == "" {
				continue
			}
			out = append(out, text)
		}
		return out
	case []string:
		out := make([]string, 0, len(item))
		for _, nested := range item {
			nested = strings.TrimSpace(nested)
			if nested == "" {
				continue
			}
			out = append(out, nested)
		}
		return out
	case string:
		text := strings.TrimSpace(item)
		if text == "" {
			return nil
		}
		return []string{text}
	default:
		return nil
	}
}

func readRealmRoles(claims map[string]any) []string {
	realmAccess, ok := claims["realm_access"].(map[string]any)
	if !ok {
		return nil
	}
	roles, ok := realmAccess["roles"].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(roles))
	for _, role := range roles {
		item, ok := role.(string)
		if !ok {
			continue
		}
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		out = append(out, item)
	}
	return out
}

func normalizeGroups(groups []string) []string {
	out := make([]string, 0, len(groups))
	for _, group := range groups {
		group = strings.TrimSpace(strings.TrimPrefix(group, "/"))
		if group == "" {
			continue
		}
		out = append(out, strings.ToLower(group))
	}
	return out
}
