package commands

import (
	"context"
	"log/slog"
	"time"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/nats/client"
	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dboxed/dboxed-volume/pkg/webdavproxy"
)

type TakeCmd struct {
	Repo   string `help:"Specify volume repo" required:""`
	Volume string `help:"Specify volume volume" required:""`

	PrevLockUuid *string `help:"Specify previous lock uuid"`

	S3WebdavProxyListen string `name:"s3-webdav-proxy-listen" help:"Specify Webdav/S3 proxy listen address" default:"127.0.0.1:0"`
}

func (cmd *TakeCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	nc, err := g.ConnectNats()
	if err != nil {
		return err
	}
	c, err := client.New(nc)
	if err != nil {
		return err
	}

	repoUuid, volumeUuid, err := getVolumeUuid(c, cmd.Repo, cmd.Volume)
	if err != nil {
		return err
	}

	s3WebdavProxy, err := webdavproxy.NewProxy(nc, repoUuid, cmd.S3WebdavProxyListen)
	if err != nil {
		return err
	}
	err = s3WebdavProxy.Start(ctx)
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

	for {
		slog.Info("refreshing volume lock")
		lockRequest.PrevLockUuid = &lockReply.Volume.Lock.Uuid
		newLockReply, err := c.VolumeLock(lockRequest)
		if err != nil {
			slog.Error("error in VolumeLock", slog.Any("error", err))
		} else {
			lockReply = newLockReply
			slog.Info("lock refreshed")
		}

		time.Sleep(15 * time.Second)
	}
}
