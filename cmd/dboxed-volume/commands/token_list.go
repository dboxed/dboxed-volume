package commands

import (
	"context"
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
)

type TokenListCmd struct{}

func (cmd *TokenListCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("", nil)
	if err != nil {
		return err
	}

	tokens, err := c.ListTokens(ctx)
	if err != nil {
		return err
	}

	for _, token := range tokens {
		slog.Info("token", slog.Any("id", token.ID), slog.Any("name", token.Name), slog.Any("created_at", token.CreatedAt))
	}

	return nil
}