package volume_backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/dboxed/dboxed-volume/pkg/util"
	"github.com/dboxed/dboxed-volume/pkg/volume"
	"github.com/dboxed/dboxed-volume/pkg/webdavproxy"
	"github.com/nats-io/nats.go"
)

type VolumeBackup struct {
	Volume   *volume.Volume
	NatsConn *nats.Conn

	RepositoryUuid        string
	SnapshotMount         string
	WebdavProxyListenAddr string
}

func (vb *VolumeBackup) Backup(ctx context.Context) error {
	snapshotName := "_backup"

	_, _ = util.RunCommand(false, "sync")

	err := vb.Volume.CreateSnapshot(snapshotName, true)
	if err != nil {
		return err
	}
	defer func() {
		err := vb.Volume.DeleteSnapshot(snapshotName)
		if err != nil {
			slog.ErrorContext(ctx, "backup snapshot deletion failed", slog.Any("error", err))
		}
	}()

	err = vb.Volume.MountSnapshot(snapshotName, vb.SnapshotMount)
	if err != nil {
		return err
	}
	defer func() {
		_, err := util.RunCommand(false, "umount", vb.SnapshotMount)
		if err != nil {
			slog.Error("deferred unmounting failed", slog.Any("error", err))
		}
	}()

	webdavProxy, err := webdavproxy.NewProxy(vb.NatsConn, vb.RepositoryUuid, vb.WebdavProxyListenAddr)
	if err != nil {
		return err
	}
	wdpAddr, err := webdavProxy.Start(ctx)
	if err != nil {
		return err
	}
	defer webdavProxy.Stop()

	rusticConfigFile, err := vb.buildRusticConfigFile(wdpAddr.String())
	if err != nil {
		return err
	}
	defer os.Remove(rusticConfigFile)

	rusticArgs := []string{"backup", vb.SnapshotMount}
	_, err = util.RunCommand(false, "rustic", rusticArgs...)
	if err != nil {
		return err
	}
	return nil
}

func (vb *VolumeBackup) buildRusticConfigFile(webdavAddr string) (string, error) {
	tmpFile, err := os.CreateTemp("", "")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	config := fmt.Sprintf(`
[repository]
repository = "opendal:webdav"
password = "test"
[repository.options]
endpoint = "http://%s"
`, webdavAddr)

	_, err = tmpFile.Write([]byte(config))
	if err != nil {
		return "", err
	}

	return tmpFile.Name(), nil
}
