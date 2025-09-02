package volume_backup

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/util"
	"github.com/dboxed/dboxed-volume/pkg/volume"
	"github.com/dboxed/dboxed-volume/pkg/webdavproxy"
	"github.com/pelletier/go-toml/v2"
)

type VolumeBackup struct {
	Client *client.Client
	Volume *volume.Volume

	RepositoryId          int64
	RusticPassword        string
	SnapshotMount         string
	WebdavProxyListenAddr string
}

func (vb *VolumeBackup) Backup(ctx context.Context) error {
	snapshotName := "_backup"

	_ = util.RunCommand("sync")

	err := vb.Volume.UnmountSnapshot(snapshotName)
	if err != nil {
		return err
	}

	err = vb.Volume.CreateSnapshot(snapshotName, true)
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
		err := vb.Volume.UnmountSnapshot(snapshotName)
		if err != nil {
			slog.Error("deferred unmounting failed", slog.Any("error", err))
		}
	}()

	fs := webdavproxy.NewFileSystem(ctx, vb.Client, vb.RepositoryId)

	webdavProxy, err := webdavproxy.NewProxy(fs, vb.WebdavProxyListenAddr)
	if err != nil {
		return err
	}
	wdpAddr, err := webdavProxy.Start(ctx)
	if err != nil {
		return err
	}
	defer webdavProxy.Stop()

	configDir, err := vb.buildRusticConfigDir(wdpAddr.String())
	if err != nil {
		return err
	}
	defer os.RemoveAll(configDir)

	rusticArgs := []string{"backup", "--init", vb.SnapshotMount}
	c := util.CommandHelper{
		Command: "rustic",
		Args:    rusticArgs,
		Dir:     configDir,
	}
	err = c.Run()
	if err != nil {
		return err
	}
	return nil
}

func (vb *VolumeBackup) buildRusticConfigDir(webdavAddr string) (string, error) {
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		return "", err
	}
	doRm := true
	defer func() {
		if doRm {
			_ = os.RemoveAll(tmpDir)
		}
	}()

	config := RusticConfig{
		Repository: RusticConfigRepository{
			Repository: "opendal:webdav",
			Password:   vb.RusticPassword,
			Options: RusticConfigRepositoryOptions{
				Endpoint: fmt.Sprintf("http://%s", webdavAddr),
			},
		},
	}
	configBytes, err := toml.Marshal(config)
	if err != nil {
		return "", err
	}

	err = os.WriteFile(filepath.Join(tmpDir, "rustic.toml"), configBytes, 0600)
	if err != nil {
		return "", err
	}

	doRm = false
	return tmpDir, nil
}
