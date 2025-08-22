package volume

import (
	"fmt"

	"github.com/dboxed/dboxed-volume/pkg/losetup"
	"github.com/dboxed/dboxed-volume/pkg/lvm"
)

var allowedFsTypes = []string{
	"ext2",
	"ext3",
	"ext4",
	"xfs",
	"btrfs",
}

type Volume struct {
	image string

	loDev         string
	attachedLoDev bool
	fsLv          *lvm.LVEntry
	tpLv          *lvm.LVEntry
}

func Open(image string) (*Volume, error) {
	loDev, attached, err := losetup.GetOrAttach(image, true)
	if err != nil {
		return nil, err
	}
	doDetach := attached
	defer func() {
		if doDetach {
			_ = losetup.Detach(loDev)
		}
	}()

	lvs, err := lvm.FindPVLVs(loDev)
	if err != nil {
		return nil, err
	}

	var fsLv *lvm.LVEntry
	var tpLv *lvm.LVEntry
	for _, lv := range lvs {
		if lv.LvTags == "fs" {
			fsLv = &lv
		}
		if lv.LvTags == "tp" {
			tpLv = &lv
		}
	}
	if fsLv == nil {
		return nil, fmt.Errorf("logical volume with fs tag not found")
	}
	if tpLv == nil {
		return nil, fmt.Errorf("logical volume with tp tag not found")
	}

	v := &Volume{
		image:         image,
		loDev:         loDev,
		attachedLoDev: attached,
		fsLv:          fsLv,
		tpLv:          tpLv,
	}

	return v, nil
}

func (v *Volume) Close(detach bool) error {
	if v.attachedLoDev {
		if detach {
			err := losetup.Detach(v.loDev)
			if err != nil {
				return err
			}
		}
		v.attachedLoDev = false
	}
	return nil
}
