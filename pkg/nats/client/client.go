package client

import (
	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dboxed/dboxed-volume/pkg/nats/nats_handlers"
	"github.com/nats-io/nats.go"
)

type Client struct {
	nc *nats.Conn
}

func New(nc *nats.Conn) (*Client, error) {
	c := &Client{
		nc: nc,
	}

	return c, nil
}

func (c *Client) RepositoryCreate(req *dproto.RepositoryCreateRequest) (*dproto.RepositoryCreateReply, error) {
	return nats_handlers.Request[*dproto.RepositoryCreateReply](c.nc, "dboxed-volume.repository.create", req)
}

func (c *Client) RepositoryUpdate(req *dproto.RepositoryUpdateRequest) (*dproto.RepositoryUpdateReply, error) {
	return nats_handlers.Request[*dproto.RepositoryUpdateReply](c.nc, "dboxed-volume.repository.update", req)
}

func (c *Client) RepositoryByName(name string) (*dproto.RepositoryByNameReply, error) {
	return nats_handlers.Request[*dproto.RepositoryByNameReply](c.nc, "dboxed-volume.repository.by-name", &dproto.RepositoryByNameRequest{
		Name: name,
	})
}

func (c *Client) VolumeCreate(req *dproto.VolumeCreateRequest) (*dproto.VolumeCreateReply, error) {
	return nats_handlers.Request[*dproto.VolumeCreateReply](c.nc, "dboxed-volume.volume.create", req)
}

func (c *Client) VolumeByName(repositoryUuid string, name string) (*dproto.VolumeByNameReply, error) {
	return nats_handlers.Request[*dproto.VolumeByNameReply](c.nc, "dboxed-volume.volume.by-name", &dproto.VolumeByNameRequest{
		RepositoryUuid: repositoryUuid,
		Name:           name,
	})
}

func (c *Client) VolumeLock(req *dproto.VolumeLockRequest) (*dproto.VolumeLockReply, error) {
	return nats_handlers.Request[*dproto.VolumeLockReply](c.nc, "dboxed-volume.volume.lock", req)
}
