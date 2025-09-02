package volume

import (
	"fmt"
	"os"
	"slices"
	"strings"

	"github.com/dboxed/dboxed-volume/pkg/fallocate"
	"github.com/dboxed/dboxed-volume/pkg/losetup"
	"github.com/dboxed/dboxed-volume/pkg/lvm"
	"github.com/dboxed/dboxed-volume/pkg/util"
	"github.com/google/uuid"
)

type CreateOptions struct {
	ImagePath string
	ImageSize int64
	FsSize    int64
	FsType    string
	Force     bool

	VgName string
}

func Create(opts CreateOptions) error {
	if _, err := os.Stat(opts.ImagePath); err == nil && !opts.Force {
		return fmt.Errorf("image '%s' already exists, we won't overwrite it", opts.ImagePath)
	}

	if !slices.Contains(AllowedFsTypes, opts.FsType) {
		return fmt.Errorf("invalid fs-type, must be one of %s", strings.Join(AllowedFsTypes, ", "))
	}

	loDevs, err := losetup.List()
	if err != nil {
		return err
	}
	for _, ld := range loDevs {
		if ld.BackFile == opts.ImagePath {
			return fmt.Errorf("file %s is already attached to loop device %s", opts.ImagePath, ld.Name)
		}
	}

	vgName := opts.VgName
	if vgName == "" {
		vgName = uuid.NewString()
	}

	tpName := "thin_pool"
	volName := "fs"

	f, err := os.Create(opts.ImagePath)
	if err != nil {
		return err
	}
	defer f.Close()
	err = fallocate.Fallocate(f, 0, opts.ImageSize)
	if err != nil {
		return err
	}

	loDev, err := losetup.Attach(opts.ImagePath)
	if err != nil {
		return err
	}

	defer losetup.Detach(loDev)

	err = lvm.PVCreate(loDev)
	if err != nil {
		return err
	}

	err = lvm.VGCreate(vgName, loDev)
	if err != nil {
		return err
	}
	defer func() {
		_ = lvm.VGDeactivate(vgName)
	}()

	err = lvm.TPCreate100(vgName, tpName, []string{"tp"})
	if err != nil {
		return err
	}

	err = lvm.TLVCreate(vgName, tpName, volName, opts.FsSize, []string{"fs"})
	if err != nil {
		return err
	}

	fsDev := buildDevName(vgName, volName)
	err = util.RunCommand(fmt.Sprintf("mkfs.%s", opts.FsType), fsDev)
	if err != nil {
		return err
	}

	return nil
}
