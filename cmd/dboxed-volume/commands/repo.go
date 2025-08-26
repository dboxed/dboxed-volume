package commands

import (
	"github.com/dboxed/dboxed-volume/pkg/nats/client"
	"github.com/google/uuid"
)

type RepoCmd struct {
	Create RepoCreateCmd `cmd:"" help:"Create a repository"`
	Update RepoUpdateCmd `cmd:"" help:"Update a repository"`
}

func getRepoUuid(c *client.Client, repo string) (string, error) {
	repoUuid := repo
	_, err := uuid.Parse(repoUuid)
	if err != nil {
		rep, err := c.RepositoryByName(repo)
		if err != nil {
			return "", err
		}
		repoUuid = rep.Repository.Uuid
	}
	return repoUuid, nil
}
