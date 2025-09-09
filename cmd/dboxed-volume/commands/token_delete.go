package commands

import (
	"context"
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
)

type TokenDeleteCmd struct {
	Id string `help:"Token ID" required:""`
}

func (cmd *TokenDeleteCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New(g.ApiUrl, g.ApiToken)
	if err != nil {
		return err
	}

	err = c.DeleteToken(ctx, cmd.Id)
	if err != nil {
		return err
	}

	slog.Info("token deleted", slog.Any("id", cmd.Id))

	return nil
}
