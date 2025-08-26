package commands

import (
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/nats/client"
	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dustin/go-humanize"
)

type VolumeCreateCmd struct {
	Repo string `help:"Specify the dboxed-volume repo" required:""`

	Name   string `help:"Specify the volume name. Must be unique in the repository."`
	FsType string `help:"Specify the filesystem type" default:"ext4"`
	FsSize string `help:"Specify the maximum filesystem size." required:""`
}

func (cmd *VolumeCreateCmd) Run(g *flags.GlobalFlags) error {
	nc, err := g.ConnectNats()
	if err != nil {
		return err
	}
	c, err := client.New(nc)
	if err != nil {
		return err
	}

	fsSize, err := humanize.ParseBytes(cmd.FsSize)
	if err != nil {
		return err
	}

	repoUuid, err := getRepoUuid(c, cmd.Repo)
	if err != nil {
		return err
	}

	rep, err := c.VolumeCreate(&dproto.VolumeCreateRequest{
		RepositoryUuid: repoUuid,
		Name:           cmd.Name,
		FsSize:         int64(fsSize),
		FsType:         cmd.FsType,
	})
	if err != nil {
		return err
	}

	slog.Info("new volume created", slog.Any("uuid", rep.Volume.Uuid))

	return nil
}
