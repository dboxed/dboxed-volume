package commands

import (
	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/volume"
)

type MountCmd struct {
	Image      string `help:"Specify the location of the volume image" type:"existingfile"`
	MountPoint string `help:"Specify the mount point" type:"existingdir"`
}

func (cmd *MountCmd) Run(g *flags.GlobalFlags) error {
	v, err := volume.Open(cmd.Image)
	if err != nil {
		return err
	}
	defer v.Close(true)

	err = v.Mount(cmd.MountPoint)
	if err != nil {
		return err
	}

	err = v.Close(false)
	if err != nil {
		return err
	}

	return nil
}
