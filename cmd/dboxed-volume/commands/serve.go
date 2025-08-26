package commands

import (
	"context"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/nats/server"
	"github.com/dboxed/dboxed-volume/pkg/repository"
	"github.com/dboxed/dboxed-volume/pkg/s3proxy"
)

type ServeCmd struct {
	NatsReplicas int `help:"Specify nats replicas to use for kv stores" default:"2"`
}

func (cmd *ServeCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	nc, err := g.ConnectNats()
	if err != nil {
		return err
	}

	rs, err := repository.OpenStore(nc, cmd.NatsReplicas)
	if err != nil {
		return err
	}

	s, err := server.New(nc, rs)
	if err != nil {
		return err
	}
	err = s.Start(ctx)
	if err != nil {
		return err
	}

	wp, err := s3proxy.NewNatsService(ctx, nc, rs)
	if err != nil {
		return err
	}
	err = wp.Start()
	if err != nil {
		return err
	}

	<-ctx.Done()

	return nil
}
