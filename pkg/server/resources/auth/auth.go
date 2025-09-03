package auth

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"reflect"
	"slices"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/danielgtaylor/huma/v2"
	"github.com/dboxed/dboxed-common/db/querier"
	"github.com/dboxed/dboxed-common/huma_utils"
	"github.com/dboxed/dboxed-common/util"
	"github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
	"github.com/dboxed/dboxed-volume/pkg/server/huma_metadata"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/golang-jwt/jwt/v5"
)

const TokenPrefix = "dvt_"

type AuthHandler struct {
	config config.Config

	oidcProvider       *oidc.Provider
	oidcProviderClaims map[string]any
}

func NewAuthHandler(config config.Config) *AuthHandler {
	h := &AuthHandler{
		config: config,
	}

	return h
}

func (s *AuthHandler) Init(ctx context.Context, api huma.API) error {

	if s.config.Auth.OidcIssuerUrl == "" {
		return fmt.Errorf("missing oidc issuer url")
	}

	provider, err := oidc.NewProvider(ctx, s.config.Auth.OidcIssuerUrl)
	if err != nil {
		return err
	}
	s.oidcProvider = provider

	err = provider.Claims(&s.oidcProviderClaims)
	if err != nil {
		return err
	}

	huma.Get(api, "/v1/auth/info", s.restInfo, huma_utils.MetadataModifier(huma_metadata.SkipAuth, true))
	huma.Get(api, "/v1/auth/me", s.restMe)

	return nil
}

func (s *AuthHandler) restInfo(ctx context.Context, input *struct{}) (*huma_utils.JsonBody[models.AuthInfo], error) {
	ret := models.AuthInfo{
		OidcIssuerUrl: s.config.Auth.OidcIssuerUrl,
		OidcClientId:  s.config.Auth.OidcClientId,
	}
	return huma_utils.NewJsonBody(ret), nil
}

func (s *AuthHandler) restMe(ctx context.Context, input *struct{}) (*huma_utils.JsonBody[models.User], error) {
	return huma_utils.NewJsonBody(MustGetUser(ctx)), nil
}

// verifyIDToken verifies that an *oauth2.Token is a valid *oidc.IDToken.
func (s *AuthHandler) verifyIDToken(ctx context.Context, rawIDToken string) (*oidc.IDToken, error) {
	oidcConfig := &oidc.Config{
		ClientID: s.config.Auth.OidcClientId,
	}

	return s.oidcProvider.Verifier(oidcConfig).Verify(ctx, rawIDToken)
}

func getClaimValue[T any](m jwt.MapClaims, n string, missingOk bool) (T, error) {
	var z T
	i, ok := m[n]
	if !ok {
		if missingOk {
			return z, nil
		}
		return z, fmt.Errorf("missing %s claim", n)
	}
	v, ok := i.(T)
	if !ok {
		return z, fmt.Errorf("invalid %s claim", n)
	}
	return v, nil
}

func (s *AuthHandler) buildUserFromIDToken(idToken *oidc.IDToken) (*models.User, error) {
	var claims jwt.MapClaims
	err := idToken.Claims(&claims)
	if err != nil {
		return nil, err
	}

	sub, err := claims.GetSubject()
	if err != nil {
		return nil, err
	}
	email, err := getClaimValue[string](claims, "email", false)
	if err != nil {
		return nil, err
	}
	name, err := getClaimValue[string](claims, "name", false)
	if err != nil {
		return nil, err
	}

	avatar, err := getClaimValue[string](claims, "avatar", true)
	if err != nil {
		return nil, err
	}

	isAdmin := false
	if slices.Contains(s.config.Auth.AdminUsers, sub) {
		isAdmin = true
	}

	return &models.User{
		ID:      sub,
		EMail:   email,
		Name:    name,
		Avatar:  avatar,
		IsAdmin: isAdmin,
	}, nil
}

func (s *AuthHandler) AuthMiddleware(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		if huma_utils.HasMetadataTrue(ctx, huma_metadata.SkipAuth) {
			next(ctx)
			return
		}
		noToken := huma_utils.HasMetadataTrue(ctx, huma_metadata.SkipAuth)

		authz, err := GetAuthorizationToken(ctx)
		if err != nil {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, err.Error(), err)
			return
		}

		var user *models.User
		if strings.HasPrefix(authz, TokenPrefix) {
			if noToken {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, err.Error(), err)
				return
			}
			user, err = s.checkDboxedToken(ctx, authz)
			if err != nil {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, err.Error(), err)
				return
			}
		} else {
			user, err = s.checkOidcToken(ctx, authz)
			if err != nil {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, err.Error(), err)
				return
			}
			err = s.updateDBUser(ctx, user)
			if err != nil {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, err.Error(), err)
				return
			}
		}

		ctx = huma.WithValue(ctx, "user", user)

		if huma_utils.HasMetadataTrue(ctx, huma_metadata.NeedAdmin) {
			if !user.IsAdmin {
				_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "must be admin")
				return
			}
		}

		next(ctx)
	}
}

func (s *AuthHandler) checkDboxedToken(ctx huma.Context, authz string) (*models.User, error) {
	q := querier.GetQuerier(ctx.Context())
	t, err := dmodel.GetTokenByToken(q, authz)
	if err != nil {
		return nil, err
	}
	isAdmin := false
	if slices.Contains(s.config.Auth.AdminUsers, t.UserID) {
		isAdmin = true
	}
	u := models.UserFromDB(*t.User, isAdmin)
	return &u, nil
}

func (s *AuthHandler) checkOidcToken(ctx huma.Context, authz string) (*models.User, error) {
	idToken, err := s.verifyIDToken(ctx.Context(), authz)
	if err != nil {
		return nil, err
	}
	user, err := s.buildUserFromIDToken(idToken)
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *AuthHandler) updateDBUser(ctx huma.Context, user *models.User) error {
	q := querier.GetQuerier(ctx.Context())

	newDbUser := dmodel.User{
		ID:     user.ID,
		Name:   user.Name,
		Email:  user.EMail,
		Avatar: user.Avatar,
	}

	needUpdate := false
	dbUser, err := dmodel.GetUserById(q, user.ID)
	if err != nil {
		if !util.IsSqlNotFoundError(err) {
			return err
		}
		needUpdate = true
	} else {
		needUpdate = !reflect.DeepEqual(models.UserFromDB(*dbUser, user.IsAdmin), *user)
	}
	if !needUpdate {
		return nil
	}
	slog.InfoContext(ctx.Context(), "updating user in DB", slog.Any("user", *user))

	err = newDbUser.CreateOrUpdate(q)
	if err != nil {
		return err
	}
	return nil
}

func GetUser(ctx context.Context) *models.User {
	userI := ctx.Value("user")
	if userI == nil {
		return nil
	}
	user, ok := userI.(*models.User)
	if !ok {
		return nil
	}
	return user
}

func MustGetUser(ctx context.Context) models.User {
	user := GetUser(ctx)
	if user == nil {
		panic("missing user")
	}
	return *user
}

func EnsureAdmin(c context.Context) error {
	u := MustGetUser(c)
	if !u.IsAdmin {
		return huma.Error401Unauthorized("must be an admin")
	}
	return nil
}
