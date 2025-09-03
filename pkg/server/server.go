package server

import (
	"context"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/danielgtaylor/huma/v2"
	"github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/auth"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/healthz"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/repositories"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/s3proxy"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/tokens"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/users"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/volumes"
	"github.com/gin-gonic/gin"

	_ "github.com/mattn/go-sqlite3"
)

type DboxedVolumeServer struct {
	config config.Config

	oidcProvider *oidc.Provider

	ginEngine  *gin.Engine
	api        huma.API
	humaConfig huma.Config

	healthz      *healthz.HealthzServer
	auth         *auth.AuthHandler
	users        *users.Users
	tokens       *tokens.Tokens
	repositories *repositories.Repositories
	volumes      *volumes.Volumes
	s3proxy      *s3proxy.S3Proxy
}

func NewDboxedVolumeServer(ctx context.Context, config config.Config) (*DboxedVolumeServer, error) {
	s := &DboxedVolumeServer{
		config: config,
	}

	var err error
	if config.Auth.OidcIssuerUrl != "" {
		s.oidcProvider, err = oidc.NewProvider(ctx, config.Auth.OidcIssuerUrl)
		if err != nil {
			return nil, err
		}
	}

	s.healthz = healthz.New(config)
	s.auth = auth.NewAuthHandler(config)
	s.users = users.New()
	s.tokens = tokens.New()
	s.repositories = repositories.New(config)
	s.volumes = volumes.New(config)
	s.s3proxy = s3proxy.New(config)

	return s, nil
}

func (s *DboxedVolumeServer) InitApi(ctx context.Context) error {
	var err error

	s.api.UseMiddleware(s.auth.AuthMiddleware(s.api))

	err = s.healthz.Init(s.api)
	if err != nil {
		return err
	}

	err = s.auth.Init(ctx, s.api)
	if err != nil {
		return err
	}

	err = s.users.Init(s.api)
	if err != nil {
		return err
	}

	err = s.tokens.Init(s.api)
	if err != nil {
		return err
	}

	err = s.repositories.Init(s.api)
	if err != nil {
		return err
	}

	err = s.volumes.Init(s.api)
	if err != nil {
		return err
	}

	err = s.s3proxy.Init(s.api)
	if err != nil {
		return err
	}

	return nil
}
