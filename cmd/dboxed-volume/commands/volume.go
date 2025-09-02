package commands

import (
	"context"
	"strconv"

	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
)

type VolumeCmd struct {
	Create VolumeCreateCmd `cmd:"" help:"Create a volume in the repository"`
	Serve  VolumeServeCmd  `cmd:"" help:"Lock, mount and sync a volume"`
}

func getVolume(ctx context.Context, c *client.Client, repo string, volume string) (*models.Repository, *models.Volume, error) {
	r, err := getRepo(ctx, c, repo)
	if err != nil {
		return nil, nil, err
	}

	volumeId, err := strconv.ParseInt(volume, 10, 64)
	if err == nil {
		v, err := c.GetVolumeById(ctx, r.ID, volumeId)
		if err != nil {
			return nil, nil, err
		}
		return r, v, nil
	} else {
		v, err := c.GetVolumeByName(ctx, r.ID, volume)
		if err != nil {
			return nil, nil, err
		}
		return r, v, nil
	}
}
