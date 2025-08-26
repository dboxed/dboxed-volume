package commands

import (
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/nats/client"
	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
)

type RepoUpdateCmd struct {
	Repo string `help:"Specify the repository." required:""`

	S3Endpoint        string  `name:"s3-endpoint" help:"Specify S3 endpoint" default:"s3.amazonaws.com"`
	S3Region          *string `name:"s3-region" help:"Specify S3 region" optional:""`
	S3Bucket          string  `name:"s3-bucket" help:"Specify S3 bucket" required:""`
	S3AccessKeyId     string  `name:"s3-access-key-id" help:"Specify S3 access key id" required:""`
	S3SecretAccessKey string  `name:"s3-secret-access-key" help:"Specify S3 secret access key" required:""`
}

func (cmd *RepoUpdateCmd) Run(g *flags.GlobalFlags) error {
	nc, err := g.ConnectNats()
	if err != nil {
		return err
	}
	c, err := client.New(nc)
	if err != nil {
		return err
	}

	repoUuid, err := getRepoUuid(c, cmd.Repo)
	if err != nil {
		return err
	}

	req := &dproto.RepositoryUpdateRequest{
		Uuid: repoUuid,
	}

	req.S3 = &dproto.RepositoryConfigS3{
		Endpoint:        cmd.S3Endpoint,
		Region:          cmd.S3Region,
		Bucket:          cmd.S3Bucket,
		AccessKeyId:     cmd.S3AccessKeyId,
		SecretAccessKey: cmd.S3SecretAccessKey,
	}

	rep, err := c.RepositoryUpdate(req)
	if err != nil {
		return err
	}

	slog.Info("repository updated", slog.Any("uuid", rep.Repository.Uuid))

	return nil
}
