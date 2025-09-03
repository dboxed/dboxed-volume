package models

import (
	"time"

	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
)

type Repository struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	Uuid string `json:"uuid"`

	S3 *RepositoryStorageS3 `json:"s3"`

	Rustic *RepositoryBackupRustic `json:"rustic"`
}

type RepositoryStorageS3 struct {
	Endpoint string  `json:"endpoint"`
	Region   *string `json:"region"`
	Bucket   string  `json:"bucket"`
	Prefix   string  `json:"prefix"`
}

type RepositoryBackupRustic struct {
	Password string `json:"password"`
}

type CreateRepository struct {
	Name string `json:"name"`

	S3 *CreateRepositoryStorageS3 `json:"s3"`

	Rustic *CreateRepositoryBackupRustic `json:"rustic"`
}

type CreateRepositoryStorageS3 struct {
	Endpoint        string  `json:"endpoint"`
	Region          *string `json:"region"`
	Bucket          string  `json:"bucket"`
	Prefix          string  `json:"prefix"`
	AccessKeyId     string  `json:"accessKeyId"`
	SecretAccessKey string  `json:"secretAccessKey"`
}

type CreateRepositoryBackupRustic struct {
	Password string `json:"password"`
}

type UpdateRepository struct {
	S3     *UpdateRepositoryStorageS3    `json:"s3"`
	Rustic *UpdateRepositoryBackupRustic `json:"rustic"`
}

type UpdateRepositoryStorageS3 struct {
	Endpoint        *string `json:"endpoint,omitempty"`
	Region          *string `json:"region,omitempty"`
	Bucket          *string `json:"bucket,omitempty"`
	Prefix          *string `json:"prefix,omitempty"`
	AccessKeyId     *string `json:"accessKeyId,omitempty"`
	SecretAccessKey *string `json:"secretAccessKey,omitempty"`
}

type UpdateRepositoryBackupRustic struct {
	Password *string `json:"password,omitempty"`
}

func RepositoryFromDB(v dmodel.Repository) Repository {
	ret := Repository{
		ID:        v.ID,
		CreatedAt: v.CreatedAt,
		Uuid:      v.Uuid,
	}
	if v.S3 != nil {
		ret.S3 = &RepositoryStorageS3{
			Endpoint: v.S3.Endpoint.V,
			Region:   v.S3.Region,
			Bucket:   v.S3.Bucket.V,
			Prefix:   v.S3.Prefix.V,
		}
	}
	if v.Rustic != nil {
		ret.Rustic = &RepositoryBackupRustic{
			Password: v.Rustic.Password.V,
		}
	}
	return ret
}
