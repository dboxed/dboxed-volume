package commands

import (
	"github.com/dboxed/dboxed-volume/pkg/nats/client"
	"github.com/google/uuid"
)

type VolumeCmd struct {
	Create VolumeCreateCmd `cmd:"" help:"Create a volume in the repository"`
	Take   TakeCmd         `cmd:"" help:"Take, mount and sync a volume"`
}

func getVolumeUuid(c *client.Client, repo string, volume string) (string, string, error) {
	repoUuid, err := getRepoUuid(c, repo)
	if err != nil {
		return "", "", err
	}

	volumeUuid := volume
	_, err = uuid.Parse(volumeUuid)
	if err != nil {
		rep, err := c.VolumeByName(repoUuid, volume)
		if err != nil {
			return "", "", err
		}
		volumeUuid = rep.Volume.Uuid
	}
	return repoUuid, volumeUuid, nil
}
