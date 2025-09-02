package commands

import (
	"context"
	"strconv"

	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
)

type RepoCmd struct {
	Create RepoCreateCmd `cmd:"" help:"Create a repository"`
	Update RepoUpdateCmd `cmd:"" help:"Update a repository"`
}

func getRepo(ctx context.Context, c *client.Client, repo string) (*models.Repository, error) {
	repoId, err := strconv.ParseInt(repo, 10, 64)
	if err == nil {
		return c.GetRepositoryById(ctx, repoId)
	} else {
		return c.GetRepositoryByName(ctx, repo)
	}
}
