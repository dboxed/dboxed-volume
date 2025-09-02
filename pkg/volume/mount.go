package volume

import (
	"log/slog"

	"github.com/dboxed/dboxed-volume/pkg/util"
	"github.com/moby/sys/mountinfo"
)

func (v *Volume) Mount(mountTarget string) error {
	lvDev := v.DevName()

	mounts, err := mountinfo.GetMounts(nil)
	if err != nil {
		return err
	}

	for _, m := range mounts {
		if m.Mountpoint == mountTarget && m.Source == lvDev {
			slog.Info("volume already mounted", slog.Any("mountPoint", m.Mountpoint), slog.Any("source", m.Source))
			return nil
		}
	}

	err = util.RunCommand("mount", lvDev, mountTarget)
	if err != nil {
		return err
	}

	return nil
}
