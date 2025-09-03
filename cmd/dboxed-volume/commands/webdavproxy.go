package commands

import (
	"context"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/webdavproxy"
)

type WebdavProxyCmd struct {
	Repo string `help:"Specify the dboxed-volume repo" required:""`

	WebdavProxyListen string `help:"Specify Webdav/S3 proxy listen address" default:"127.0.0.1:10000"`
}

func (cmd *WebdavProxyCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("", nil)
	if err != nil {
		return err
	}

	r, err := getRepo(ctx, c, cmd.Repo)
	if err != nil {
		return err
	}

	fs := webdavproxy.NewFileSystem(ctx, c, r.ID)

	webdavProxy, err := webdavproxy.NewProxy(fs, cmd.WebdavProxyListen)
	if err != nil {
		return err
	}
	_, err = webdavProxy.Start(ctx)
	if err != nil {
		return err
	}
	defer webdavProxy.Stop()

	<-ctx.Done()

	return nil
}
