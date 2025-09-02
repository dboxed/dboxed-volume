package commands

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/dboxed/dboxed-volume/pkg/volume"
	"github.com/dboxed/dboxed-volume/pkg/volume_backup"
	"github.com/dustin/go-humanize"
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

	lockRequest := models.VolumeLockRequest{
		PrevLockId: cmd.PrevLockId,
	}

	slog.Info("locking volume")
	lockReply, err := c.VolumeLock(ctx, v.RepositoryID, v.ID, lockRequest)
	if err != nil {
		return err
	}
	slog.Info("volume locked", slog.Any("lockId", lockReply.LockId))

	go periodicRefreshLock(ctx, c, &lockRequest, &lockReply)

	if _, err := os.Stat(cmd.Image); err != nil {
		imageSize := v.FsSize * 2
		slog.Info("creating local volume image",
			slog.Any("path", cmd.Image),
			slog.Any("imageSize", humanize.Bytes(uint64(imageSize))),
			slog.Any("fsSize", humanize.Bytes(uint64(v.FsSize))),
			slog.Any("fsType", v.FsType),
		)
		err := volume.Create(volume.CreateOptions{
			ImagePath: cmd.Image,
			ImageSize: imageSize,
			FsSize:    v.FsSize,
			FsType:    v.FsType,
		})
		if err != nil {
			return err
		}
	}

	slog.Info("opening local volume image",
		slog.Any("path", cmd.Image),
	)
	localVolume, err := volume.Open(cmd.Image)
	if err != nil {
		return err
	}

	slog.Info("mounting volume", slog.Any("mountPath", cmd.Mount))
	err = localVolume.Mount(cmd.Mount)
	if err != nil {
		return err
	}

	vb := volume_backup.VolumeBackup{
		Client:                c,
		Volume:                localVolume,
		RepositoryId:          v.RepositoryID,
		RusticPassword:        r.Rustic.Password,
		SnapshotMount:         cmd.SnapshotMount,
		WebdavProxyListenAddr: cmd.WebdavProxyListen,
	}

	for {
		select {
		case <-time.After(backupInterval):
			err := vb.Backup(ctx)
			if err != nil {
				slog.Error("backup failed", slog.Any("error", err))
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
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
