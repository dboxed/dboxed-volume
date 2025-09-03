package commands

import (
	"context"
	"os"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"sigs.k8s.io/yaml"
)

type VolumeListCmd struct {
	Repo string `help:"Specify the dboxed-volume repo"`
}

type repoWithVolumes struct {
	models.Repository
	Volumes []models.Volume `json:"volumes"`
}

func (cmd *VolumeListCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("", nil)
	if err != nil {
		return err
	}

	var y any
	if cmd.Repo != "" {
		r, err := getRepo(ctx, c, cmd.Repo)
		if err != nil {
			return err
		}

		volumes, err := c.ListVolumes(ctx, r.ID)
		if err != nil {
			return err
		}

		y = volumes
	} else {
		var l []repoWithVolumes
		repos, err := c.ListRepositories(ctx)
		if err != nil {
			return err
		}
		for _, r := range repos {
			volumes, err := c.ListVolumes(ctx, r.ID)
			if err != nil {
				return err
			}
			l = append(l, repoWithVolumes{
				Repository: r,
				Volumes:    volumes,
			})
		}
		y = l
	}

	b, err := yaml.Marshal(y)
	if err != nil {
		return err
	}

	_, err = os.Stdout.Write(b)
	if err != nil {
		return err
	}

	return nil
}
