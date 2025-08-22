package commands

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/volume"
	"github.com/dustin/go-humanize"
)

type InitCmd struct {
	Image     string `help:"Specify the location of the volume image" type:"path"`
	ImageSize string `help:"Specify the image size (supports human readable units, e.g. 1G, 512Mi, ...)"`

	FsType string `help:"Specify the filesystem type" default:"ext4"`
	FsSize string `help:"Specify the maximum filesystem size. Defaults to 45% of the image size." optional:""`

	VgName string `help:"Specify the volume group name that. Defaults to a random name."`

	Force bool `help:"Force overwriting of existing images" short:"f"`
}

func (cmd *InitCmd) Run(g *flags.GlobalFlags) error {
	if cmd.Image == "-" {
		return fmt.Errorf("invalid image path")
	}
	if _, err := os.Stat(cmd.Image); err == nil && !cmd.Force {
		return fmt.Errorf("image '%s' already exists, we won't overwrite it", cmd.Image)
	}

	size, err := humanize.ParseBytes(cmd.ImageSize)
	if err != nil {
		return err
	}
	var fsSize uint64
	if cmd.FsSize == "" {
		fsSize = (size * 45) / 100
		// align to sector size
		fsSize = ((fsSize + 511) / 512) * 512
	} else {
		fsSize, err = humanize.ParseBytes(cmd.FsSize)
		if err != nil {
			return err
		}
		if fsSize%512 != 0 {
			return fmt.Errorf("fs-size is not a multiple of 512")
		}
	}

	slog.Info(fmt.Sprintf("creating volume with total size of %s and fs-size of %s", humanize.Bytes(size), humanize.Bytes(fsSize)))

	err = volume.Create(volume.CreateOptions{
		ImagePath: cmd.Image,
		ImageSize: int64(size),
		FsSize:    int64(fsSize),
		FsType:    cmd.FsType,
		Force:     cmd.Force,
		VgName:    cmd.VgName,
	})
	if err != nil {
		return err
	}

	return nil
}
