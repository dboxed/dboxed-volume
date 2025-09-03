package commands

import (
	"context"
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
)

type TokenGetCmd struct {
	Id string `help:"Token ID" required:""`
}

func (cmd *TokenGetCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("", nil)
	if err != nil {
		return err
	}

	token, err := getToken(ctx, c, cmd.Id)
	if err != nil {
		return err
	}

	slog.Info("token", slog.Any("id", token.ID), slog.Any("name", token.Name), slog.Any("created_at", token.CreatedAt))

	return nil
}
