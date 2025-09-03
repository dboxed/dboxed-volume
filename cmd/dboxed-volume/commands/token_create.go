package commands

import (
	"context"
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
)

type TokenCreateCmd struct {
	Name string `help:"Specify the token name. Must be unique." required:""`
}

func (cmd *TokenCreateCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("", nil)
	if err != nil {
		return err
	}

	req := models.CreateToken{
		Name: cmd.Name,
	}

	token, err := c.CreateToken(ctx, req)
	if err != nil {
		return err
	}

	slog.Info("token created", slog.Any("id", token.ID), slog.Any("name", token.Name), slog.Any("token", token.TokenStr))

	return nil
}