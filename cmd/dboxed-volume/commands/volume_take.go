package commands

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/nats/client"
	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dboxed/dboxed-volume/pkg/volume"
	"github.com/dboxed/dboxed-volume/pkg/volume_backup"
	"github.com/dustin/go-humanize"
)

type TakeCmd struct {
	Repo   string `help:"Specify volume repo" required:""`
	Volume string `help:"Specify volume volume" required:""`

	PrevLockUuid *string `help:"Specify previous lock uuid"`

	Image          string `help:"Specify the location of the volume image" type:"path" required:""`
	Mount          string `help:"Specify where to mount the volume" type:"existingdir" required:""`
	SnapshotMount  string `help:"Specify where to mount the temporary backup snapshots" type:"existingdir" required:""`
	BackupInterval string `help:"Specify the backup interval" default:"5m"`

	S3WebdavProxyListen string `name:"s3-webdav-proxy-listen" help:"Specify Webdav/S3 proxy listen address" default:"127.0.0.1:0"`
}

func (cmd *TakeCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	nc, err := g.ConnectNats()
	if err != nil {
		return err
	}
	c, err := client.New(nc)
	if err != nil {
		return err
	}

	backupInterval, err := time.ParseDuration(cmd.BackupInterval)
	if err != nil {
		return err
	}

	repoUuid, volumeUuid, err := getVolumeUuid(c, cmd.Repo, cmd.Volume)
	if err != nil {
		return err
	}

	lockRequest := &dproto.VolumeLockRequest{
		RepositoryUuid: repoUuid,
		VolumeUuid:     volumeUuid,
		PrevLockUuid:   cmd.PrevLockUuid,
	}

	slog.Info("locking volume")
	lockReply, err := c.VolumeLock(lockRequest)
	if err != nil {
		return err
	}
	slog.Info("volume locked", slog.Any("lockUuid", lockReply.Volume.Lock.Uuid))

	go periodicRefreshLock(ctx, c, lockRequest, &lockReply)

	if _, err := os.Stat(cmd.Image); err != nil {
		imageSize := lockReply.Volume.FsSize * 2
		slog.Info("creating local volume image",
			slog.Any("path", cmd.Image),
			slog.Any("imageSize", humanize.Bytes(uint64(imageSize))),
			slog.Any("fsSize", humanize.Bytes(uint64(lockReply.Volume.FsSize))),
			slog.Any("fsType", lockReply.Volume.FsType),
		)
		err := volume.Create(volume.CreateOptions{
			ImagePath: cmd.Image,
			ImageSize: imageSize,
			FsSize:    lockReply.Volume.FsSize,
			FsType:    lockReply.Volume.FsType,
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
		Volume:                localVolume,
		NatsConn:              nc,
		RepositoryUuid:        repoUuid,
		SnapshotMount:         cmd.SnapshotMount,
		WebdavProxyListenAddr: cmd.S3WebdavProxyListen,
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

func periodicRefreshLock(ctx context.Context, c *client.Client, lockRequest *dproto.VolumeLockRequest, lockReply **dproto.VolumeLockReply) {
	for {
		slog.Info("refreshing volume lock")
		lockRequest.PrevLockUuid = &(*lockReply).Volume.Lock.Uuid
		newLockReply, err := c.VolumeLock(lockRequest)
		if err != nil {
			slog.Error("error in VolumeLock", slog.Any("error", err))
		} else {
			*lockReply = newLockReply
			slog.Info("lock refreshed")
		}

		select {
		case <-time.After(15 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}
