package commands

import (
	"context"
	"log/slog"
	"time"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/dboxed/dboxed-volume/pkg/volume_serve"
)

type VolumeServeCmd struct {
	Repo   string `help:"Specify volume repo" required:""`
	Volume string `help:"Specify volume volume" required:""`

	PrevLockId *string `help:"Specify previous lock id"`

	Image          string `help:"Specify the location of the volume image" type:"path" required:""`
	Mount          string `help:"Specify where to mount the volume" type:"existingdir" required:""`
	SnapshotMount  string `help:"Specify where to mount the temporary backup snapshots" type:"existingdir" required:""`
	BackupInterval string `help:"Specify the backup interval" default:"5m"`

	WebdavProxyListen string `help:"Specify Webdav/S3 proxy listen address" default:"127.0.0.1:0"`
}

func (cmd *VolumeServeCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("")
	if err != nil {
		return err
	}

	backupInterval, err := time.ParseDuration(cmd.BackupInterval)
	if err != nil {
		return err
	}

	r, v, err := getVolume(ctx, c, cmd.Repo, cmd.Volume)
	if err != nil {
		return err
	}

	vs := volume_serve.VolumeServe{
		Client:            c,
		RepositoryId:      r.ID,
		VolumeId:          v.ID,
		PrevLockId:        cmd.PrevLockId,
		Image:             cmd.Image,
		Mount:             cmd.Mount,
		SnapshotMount:     cmd.SnapshotMount,
		BackupInterval:    backupInterval,
		WebdavProxyListen: cmd.WebdavProxyListen,
	}

	err = vs.Start(ctx)
	if err != nil {
		return err
	}

	<-ctx.Done()

	return ctx.Err()
}

func periodicRefreshLock(ctx context.Context, c *client.Client, lockRequest *models.VolumeLockRequest, volumePtr **models.Volume) {
	for {
		v := *volumePtr

		slog.Info("refreshing volume lock")
		lockRequest.PrevLockId = v.LockId
		newVolume, err := c.VolumeLock(ctx, v.RepositoryID, v.ID, *lockRequest)
		if err != nil {
			slog.Error("error in VolumeLock", slog.Any("error", err))
		} else {
			*volumePtr = newVolume
			slog.Info("lock refreshed")
		}

		select {
		case <-time.After(15 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}
