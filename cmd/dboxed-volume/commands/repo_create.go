package commands

import (
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/nats/client"
	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
)

type RepoCreateCmd struct {
	Name string `help:"Specify the repository name. Must be unique." required:""`

	S3Endpoint        string  `name:"s3-endpoint" help:"Specify S3 endpoint" default:"s3.amazonaws.com"`
	S3Region          *string `name:"s3-region" help:"Specify S3 region" optional:""`
	S3Bucket          string  `name:"s3-bucket" help:"Specify S3 bucket" required:""`
	S3AccessKeyId     string  `name:"s3-access-key-id" help:"Specify S3 access key id" required:""`
	S3SecretAccessKey string  `name:"s3-secret-access-key" help:"Specify S3 secret access key" required:""`

	S3Prefix string `name:"s3-prefix" help:"Specify the s3 prefix"`
}

func (cmd *RepoCreateCmd) Run(g *flags.GlobalFlags) error {
	nc, err := g.ConnectNats()
	if err != nil {
		return err
	}
	c, err := client.New(nc)
	if err != nil {
		return err
	}

	req := &dproto.RepositoryCreateRequest{
		Name: cmd.Name,
	}

	req.S3 = &dproto.RepositoryConfigS3{
		Endpoint:        cmd.S3Endpoint,
		Region:          cmd.S3Region,
		Bucket:          cmd.S3Bucket,
		AccessKeyId:     cmd.S3AccessKeyId,
		SecretAccessKey: cmd.S3SecretAccessKey,
		Prefix:          cmd.S3Prefix,
	}

	rep, err := c.RepositoryCreate(req)
	if err != nil {
		return err
	}

	slog.Info("new repository created", slog.Any("uuid", rep.Repository.Uuid))

	return nil
}
