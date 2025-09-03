package client

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
)

type Client struct {
	url string

	clientAuth  *config.ClientAuth
	staticToken *string

	m        sync.Mutex
	provider *oidc.Provider
}

func New(url string, staticToken *string) (*Client, error) {
	c := &Client{
		url: url,
	}

	if staticToken != nil {
		c.staticToken = staticToken
	} else {
		clientAuth, err := config.ReadClientAuth()
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, err
			} else {
				clientAuth = &config.ClientAuth{}
			}
		}
		if url != "" {
			clientAuth.ApiUrl = url
		}
		c.clientAuth = clientAuth
	}

	return c, nil
}

func (c *Client) CreateRepository(ctx context.Context, req models.CreateRepository) (*models.Repository, error) {
	return requestApi[models.Repository](ctx, c, "POST", "v1/repositories", req)
}

func (c *Client) DeleteRepository(ctx context.Context, repoId int64) error {
	_, err := requestApi[models.Volume](ctx, c, "DELETE", fmt.Sprintf("v1/repositories/%d", repoId), struct{}{})
	return err
}

func (c *Client) UpdateRepository(ctx context.Context, repoId int64, req models.UpdateRepository) (*models.Repository, error) {
	return requestApi[models.Repository](ctx, c, "PATCH", fmt.Sprintf("v1/repositories/%d", repoId), req)
}

func (c *Client) GetRepositoryById(ctx context.Context, repoId int64) (*models.Repository, error) {
	return requestApi[models.Repository](ctx, c, "GET", fmt.Sprintf("v1/repositories/%d", repoId), struct{}{})
}

func (c *Client) GetRepositoryByName(ctx context.Context, name string) (*models.Repository, error) {
	return requestApi[models.Repository](ctx, c, "GET", fmt.Sprintf("v1/repositories/by-name/%s", name), struct{}{})
}

func (c *Client) CreateVolume(ctx context.Context, repoId int64, req models.CreateVolume) (*models.Volume, error) {
	return requestApi[models.Volume](ctx, c, "POST", fmt.Sprintf("v1/repositories/%d/volumes", repoId), req)
}

func (c *Client) DeleteVolume(ctx context.Context, repoId int64, volumeId int64) error {
	_, err := requestApi[models.Volume](ctx, c, "DELETE", fmt.Sprintf("v1/repositories/%d/volumes/%d", repoId, volumeId), struct{}{})
	return err
}

func (c *Client) GetVolumeById(ctx context.Context, repoId int64, volumeId int64) (*models.Volume, error) {
	return requestApi[models.Volume](ctx, c, "GET", fmt.Sprintf("v1/repositories/%d/volumes/%d", repoId, volumeId), struct{}{})
}

func (c *Client) GetVolumeByName(ctx context.Context, repoId int64, name string) (*models.Volume, error) {
	return requestApi[models.Volume](ctx, c, "GET", fmt.Sprintf("v1/repositories/%d/volumes/by-name/%s", repoId, name), struct{}{})
}

func (c *Client) VolumeLock(ctx context.Context, repoId int64, volumeId int64, req models.VolumeLockRequest) (*models.Volume, error) {
	return requestApi[models.Volume](ctx, c, "POST", fmt.Sprintf("v1/repositories/%d/volumes/%d/lock", repoId, volumeId), req)
}
