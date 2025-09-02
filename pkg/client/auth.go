package client

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"golang.org/x/oauth2"
)

func (c *Client) GetClientAuth() *config.ClientAuth {
	return c.clientAuth
}

func (c *Client) WriteClientAuth() error {
	return config.WriteClientAuth(c.clientAuth)
}

func (c *Client) Login(ctx context.Context) error {
	authInfo, err := requestApi2[models.AuthInfo](ctx, c, "GET", "v1/auth/info", struct{}{}, false)
	if err != nil {
		return err
	}

	c.m.Lock()
	defer c.m.Unlock()

	c.clientAuth.AuthInfo = authInfo
	c.clientAuth.Token = nil

	ocfg, err := c.buildOAuth2Config(ctx)
	if err != nil {
		return err
	}

	deviceAuth, err := ocfg.DeviceAuth(ctx, oauth2.AccessTypeOffline)
	if err != nil {
		return err
	}

	fmt.Printf("Please visit %s to login\nYour user code is %s\n", deviceAuth.VerificationURIComplete, deviceAuth.UserCode)

	token, err := ocfg.DeviceAccessToken(ctx, deviceAuth)
	if err != nil {
		return err
	}

	c.clientAuth.AuthInfo = authInfo
	c.clientAuth.Token = token

	err = c.WriteClientAuth()
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) buildOAuth2Config(ctx context.Context) (*oauth2.Config, error) {
	provider := c.provider
	if provider == nil {
		var err error
		provider, err = oidc.NewProvider(ctx, c.clientAuth.AuthInfo.OidcIssuerUrl)
		if err != nil {
			return nil, err
		}
		c.provider = provider
	}

	cfg := &oauth2.Config{
		ClientID: c.clientAuth.AuthInfo.OidcClientId,
		Endpoint: provider.Endpoint(),
	}

	return cfg, nil
}

func (c *Client) RefreshToken(ctx context.Context) error {
	c.m.Lock()
	defer c.m.Unlock()

	if c.clientAuth == nil || c.clientAuth.Token == nil {
		return fmt.Errorf("client has no token, please login first")
	}

	if c.clientAuth.Token.Valid() {
		return nil
	}

	slog.InfoContext(ctx, "refreshing token")

	ocfg, err := c.buildOAuth2Config(ctx)
	if err != nil {
		return err
	}
	ts := ocfg.TokenSource(ctx, c.clientAuth.Token)
	newToken, err := ts.Token()
	if err != nil {
		return err
	}

	c.clientAuth.Token = newToken
	err = c.WriteClientAuth()
	if err != nil {
		return err
	}

	return nil
}
