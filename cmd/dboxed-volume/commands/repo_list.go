package commands

import (
	"context"
	"os"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
	"sigs.k8s.io/yaml"
)

type RepoListCmd struct {
}

func (cmd *RepoListCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("", nil)
	if err != nil {
		return err
	}

	repos, err := c.ListRepositories(ctx)
	if err != nil {
		return err
	}
	
	b, err := yaml.Marshal(repos)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(b)
	if err != nil {
		return err
	}
	return nil
}
