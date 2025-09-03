package commands

import (
	"context"

	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
)

type TokenCmd struct {
	Create TokenCreateCmd `cmd:"" help:"Create a token"`
	Get    TokenGetCmd    `cmd:"" help:"Get a token"`
	List   TokenListCmd   `cmd:"" help:"List tokens"`
	Delete TokenDeleteCmd `cmd:"" help:"Delete a token"`
}

func getToken(ctx context.Context, c *client.Client, tokenId string) (*models.Token, error) {
	return c.GetTokenById(ctx, tokenId)
}