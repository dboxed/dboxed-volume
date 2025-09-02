package commands

import (
	"context"
	"log/slog"

	"github.com/dboxed/dboxed-volume/cmd/dboxed-volume/flags"
	"github.com/dboxed/dboxed-volume/pkg/client"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
)

type RepoUpdateCmd struct {
	Repo string `help:"Specify the repository." required:""`

	S3Endpoint        *string `name:"s3-endpoint" help:"Specify S3 endpoint"`
	S3Region          *string `name:"s3-region" help:"Specify S3 region"`
	S3Bucket          *string `name:"s3-bucket" help:"Specify S3 bucket"`
	S3Prefix          *string `name:"s3-prefix" help:"Specify S3 prefix"`
	S3AccessKeyId     *string `name:"s3-access-key-id" help:"Specify S3 access key id"`
	S3SecretAccessKey *string `name:"s3-secret-access-key" help:"Specify S3 secret access key"`

	RusticPassword *string `help:"Specify the password used for encryption"`
}

func (cmd *RepoUpdateCmd) Run(g *flags.GlobalFlags) error {
	ctx := context.Background()

	c, err := client.New("")
	if err != nil {
		return err
	}

	r, err := getRepo(ctx, c, cmd.Repo)
	if err != nil {
		return err
	}

	req := models.UpdateRepository{}

	req.S3 = &models.UpdateRepositoryStorageS3{
		Endpoint:        cmd.S3Endpoint,
		Region:          cmd.S3Region,
		Bucket:          cmd.S3Bucket,
		Prefix:          cmd.S3Prefix,
		AccessKeyId:     cmd.S3AccessKeyId,
		SecretAccessKey: cmd.S3SecretAccessKey,
	}

	req.Rustic = &models.UpdateRepositoryBackupRustic{
		Password: cmd.RusticPassword,
	}

	rep, err := c.UpdateRepository(ctx, r.ID, req)
	if err != nil {
		return err
	}

	slog.Info("repository updated", slog.Any("id", rep.ID), slog.Any("uuid", rep.Uuid))

	return nil
}
