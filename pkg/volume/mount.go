package volume

import (
	"path/filepath"

	"github.com/dboxed/dboxed-volume/pkg/util"
)

func (v *Volume) Mount(mountTarget string) error {
	lvDev := filepath.Join("/dev", v.fsLv.VgName, v.fsLv.LvName)
	_, err := util.RunCommand(false, "mount", lvDev, mountTarget)
	if err != nil {
		return err
	}

	return nil
}
