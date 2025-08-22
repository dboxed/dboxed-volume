package volume

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"syscall"

	"github.com/dboxed/dboxed-volume/pkg/lvm"
	"github.com/dboxed/dboxed-volume/pkg/util"
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

	syscall.Sync()

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
	lvDev := filepath.Join("/dev", v.fsLv.VgName, snapshotName)
	_, err := util.RunCommand(false, "mount", "-oro", lvDev, mountTarget)
	if err != nil {
		return err
	}
	return nil
}
