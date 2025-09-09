package commands

import (
	"context"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
)

type LoginCmd struct {
}

func (cmd *LoginCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New(g.ApiUrl, g.ApiToken)
	if err != nil {
		return err
	}

	err = c.Login(ctx)
	if err != nil {
		return err
	}

	return nil
}
