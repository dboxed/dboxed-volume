package volume_serve

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/dboxed/dboxed-volume/pkg/volume"
	"github.com/dboxed/dboxed-volume/pkg/volume_backup"
	"github.com/dustin/go-humanize"
)

type VolumeServe struct {
	Client *client.Client

	RepositoryId int64
	VolumeId     int64

	PrevLockId *string

	Image          string
	Mount          string
	SnapshotMount  string
	BackupInterval time.Duration

	WebdavProxyListen string

	log *slog.Logger

	repository *models.Repository
	volume     *models.Volume

	localVolume *volume.Volume
}

func (vs *VolumeServe) Start(ctx context.Context) error {
	vs.log = slog.With(
		slog.Any("repositoryId", vs.RepositoryId),
		slog.Any("volumeId", vs.VolumeId),
		slog.Any("mount", vs.Mount),
		slog.Any("snapshotMount", vs.SnapshotMount),
	)

	var err error
	vs.repository, err = vs.Client.GetRepositoryById(ctx, vs.RepositoryId)
	if err != nil {
		return err
	}

	err = vs.lockVolume(ctx, vs.PrevLockId)
	if err != nil {
		return err
	}

	go vs.periodicRefreshLock(ctx)

	if _, err := os.Stat(vs.Image); err != nil {
		imageSize := vs.volume.FsSize * 2
		vs.log.Info("creating local volume image",
			slog.Any("path", vs.Image),
			slog.Any("imageSize", humanize.Bytes(uint64(imageSize))),
			slog.Any("fsSize", humanize.Bytes(uint64(vs.volume.FsSize))),
			slog.Any("fsType", vs.volume.FsType),
		)
		err := volume.Create(volume.CreateOptions{
			ImagePath: vs.Image,
			ImageSize: imageSize,
			FsSize:    vs.volume.FsSize,
			FsType:    vs.volume.FsType,
		})
		if err != nil {
			return err
		}
	}

	vs.log.Info("opening local volume image",
		slog.Any("path", vs.Image),
	)
	vs.localVolume, err = volume.Open(vs.Image)
	if err != nil {
		return err
	}

	vs.log.Info("mounting volume", slog.Any("mountPath", vs.Mount))
	err = vs.localVolume.Mount(vs.Mount)
	if err != nil {
		return err
	}

	go vs.periodicBackup(ctx)

	return nil
}

func (vs *VolumeServe) lockVolume(ctx context.Context, prevLockId *string) error {
	var err error
	if prevLockId == nil {
		vs.log.Info("locking volume")
	} else {
		vs.log.Info("refreshing lock", slog.Any("prevLockId", *prevLockId))
	}
	lockRequest := models.VolumeLockRequest{
		PrevLockId: prevLockId,
	}
	vs.volume, err = vs.Client.VolumeLock(ctx, vs.RepositoryId, vs.VolumeId, lockRequest)
	if err != nil {
		return err
	}
	vs.log.Info("volume locked", slog.Any("lockId", *vs.volume.LockId))
	return nil
}

func (vs *VolumeServe) periodicBackup(ctx context.Context) {
	vb := volume_backup.VolumeBackup{
		Client:                vs.Client,
		Volume:                vs.localVolume,
		RepositoryId:          vs.repository.ID,
		RusticPassword:        vs.repository.Rustic.Password,
		SnapshotMount:         vs.SnapshotMount,
		WebdavProxyListenAddr: vs.WebdavProxyListen,
	}

	for {
		select {
		case <-time.After(vs.BackupInterval):
			err := vb.Backup(ctx)
			if err != nil {
				vs.log.Error("backup failed", slog.Any("error", err))
			}
		case <-ctx.Done():
			return
		}
	}
}

func (vs *VolumeServe) periodicRefreshLock(ctx context.Context) {
	for {
		err := vs.lockVolume(ctx, vs.volume.LockId)
		if err != nil {
			vs.log.Error("error in VolumeLock", slog.Any("error", err))
		}

		select {
		case <-time.After(15 * time.Second):
		case <-ctx.Done():
			return
		}
	}
}
