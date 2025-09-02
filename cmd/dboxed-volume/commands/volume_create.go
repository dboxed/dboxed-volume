package commands

import (
	"context"
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/dustin/go-humanize"
)

type VolumeCreateCmd struct {
	Repo string `help:"Specify the dboxed-volume repo" required:""`

	Name   string `help:"Specify the volume name. Must be unique in the repository."`
	FsType string `help:"Specify the filesystem type" default:"ext4"`
	FsSize string `help:"Specify the maximum filesystem size." required:""`
}

func (cmd *VolumeCreateCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("")
	if err != nil {
		return err
	}

	fsSize, err := humanize.ParseBytes(cmd.FsSize)
	if err != nil {
		return err
	}

	r, err := getRepo(ctx, c, cmd.Repo)
	if err != nil {
		return err
	}

	rep, err := c.CreateVolume(ctx, r.ID, models.CreateVolume{
		Name:   cmd.Name,
		FsSize: int64(fsSize),
		FsType: cmd.FsType,
	})
	if err != nil {
		return err
	}

	slog.Info("volume created", slog.Any("id", rep.ID), slog.Any("uuid", rep.Uuid))

	return nil
}
