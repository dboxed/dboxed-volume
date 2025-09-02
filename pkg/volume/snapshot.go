package volume

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/dboxed/dboxed-volume/pkg/lvm"
	"github.com/dboxed/dboxed-volume/pkg/util"
	"github.com/moby/sys/mountinfo"
)

func (v *Volume) CreateSnapshot(snapshotName string, overwrite bool) error {
	snapLv, err := lvm.LVGet(v.fsLv.VgName, snapshotName)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}
	if snapLv != nil {
		if !overwrite {
			return fmt.Errorf("snapshot %s already exists", snapshotName)
		}
		slog.Info("snapshot already exists, removing it", slog.Any("snapshotName", snapshotName))
		err = lvm.LVRemove(v.fsLv.VgName, snapshotName)
		if err != nil {
			return err
		}
	}

	_ = util.RunCommand("sync")

	slog.Info("creating snapshot", slog.Any("snapshotName", snapshotName))
	err = lvm.TLVSnapCreate(v.fsLv.VgName, v.fsLv.LvName, v.tpLv.LvName, snapshotName)
	if err != nil {
		return err
	}

	deferRemoveSnapshot := true
	defer func() {
		if deferRemoveSnapshot {
			err := lvm.LVRemove(v.fsLv.VgName, snapshotName)
			if err != nil {
				slog.Error("remove snapshot failed in defer", slog.Any("error", err))
			}
		}
	}()

	err = lvm.LVActivate(v.fsLv.VgName, snapshotName, true)
	if err != nil {
		return err
	}

	deferRemoveSnapshot = false
	return nil
}

func (v *Volume) DeleteSnapshot(snapshotName string) error {
	return lvm.LVRemove(v.fsLv.VgName, snapshotName)
}

func (v *Volume) MountSnapshot(snapshotName string, mountTarget string) error {
	lvDev := buildDevName(v.fsLv.VgName, snapshotName)
	err := util.RunCommand("mount", "-oro", lvDev, mountTarget)
	if err != nil {
		return err
	}
	return nil
}

func (v *Volume) UnmountSnapshot(snapshotName string) error {
	isMounted, err := v.IsSnapshotMounted(snapshotName)
	if err != nil {
		return err
	}
	if !isMounted {
		return nil
	}

	lvDev := buildDevName(v.fsLv.VgName, snapshotName)
	err = util.RunCommand("umount", lvDev)
	return err
}

func (v *Volume) IsSnapshotMounted(snapshotName string) (bool, error) {
	mounts, err := mountinfo.GetMounts(nil)
	if err != nil {
		return false, err
	}
	lvDev := buildDevName(v.fsLv.VgName, snapshotName)

	for _, m := range mounts {
		if m.Source == lvDev {
			return true, nil
		}
	}
	return false, nil
}
