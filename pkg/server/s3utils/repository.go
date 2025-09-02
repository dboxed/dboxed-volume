package s3utils

import (
	"fmt"
	"net/url"

	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func BuildS3ClientForRegion(r *dmodel.Repository, region string) (*minio.Client, error) {
	if r.S3 == nil {
		return nil, fmt.Errorf("not a S3 repository")
	}

	creds := credentials.NewStaticV4(r.S3.AccessKeyId.V, r.S3.SecretAccessKey.V, "")

	u, err := url.Parse(r.S3.Endpoint.V)
	if err != nil {
		return nil, err
	}
	mc, err := minio.New(u.Host, &minio.Options{
		Creds:  creds,
		Region: region,
		Secure: u.Scheme == "https",
	})
	if err != nil {
		return nil, err
	}

	return mc, nil
}

func BuildS3Client(r *dmodel.Repository) (*minio.Client, error) {
	if r.S3 == nil {
		return nil, fmt.Errorf("not a S3 repository")
	}

	region := ""
	if r.S3.Region != nil {
		region = *r.S3.Region
	}

	return BuildS3ClientForRegion(r, region)
}
