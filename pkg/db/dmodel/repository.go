package dmodel

import "github.com/dboxed/dboxed-common/db/querier"

type Repository struct {
	Base

	Name string `db:"name"`
	Uuid string `db:"uuid"`

	S3 *RepositoryStorageS3 `join:"true"`

	Rustic *RepositoryBackupRustic `join:"true""`
}

type RepositoryStorageS3 struct {
	ID querier.NullForJoin[int64] `db:"id"`

	Endpoint        querier.NullForJoin[string] `db:"endpoint"`
	Region          *string                     `db:"region"`
	Bucket          querier.NullForJoin[string] `db:"bucket"`
	Prefix          querier.NullForJoin[string] `db:"prefix"`
	AccessKeyId     querier.NullForJoin[string] `db:"access_key_id"`
	SecretAccessKey querier.NullForJoin[string] `db:"secret_access_key"`
}

type RepositoryBackupRustic struct {
	ID querier.NullForJoin[int64] `db:"id"`

	Password querier.NullForJoin[string] `db:"password"`
}

func (v *Repository) Create(q *querier.Querier) error {
	return querier.Create(q, v)
}

func (v *RepositoryStorageS3) Create(q *querier.Querier) error {
	return querier.Create(q, v)
}

func (v *RepositoryBackupRustic) Create(q *querier.Querier) error {
	return querier.Create(q, v)
}

func GetRepositoryById(q *querier.Querier, id int64, skipDeleted bool) (*Repository, error) {
	return querier.GetOne[Repository](q, map[string]any{
		"id":         id,
		"deleted_at": querier.ExcludeNonNull(skipDeleted),
	})
}

func GetRepositoryByName(q *querier.Querier, name string, skipDeleted bool) (*Repository, error) {
	return querier.GetOne[Repository](q, map[string]any{
		"name":       name,
		"deleted_at": querier.ExcludeNonNull(skipDeleted),
	})
}

func ListRepositories(q *querier.Querier, skipDeleted bool) ([]Repository, error) {
	return querier.GetMany[Repository](q, map[string]any{
		"deleted_at": querier.ExcludeNonNull(skipDeleted),
	})
}

func (v *RepositoryStorageS3) UpdateEndpoint(q *querier.Querier, endpoint string) error {
	v.Endpoint = querier.N(endpoint)
	return querier.UpdateOneFromStruct(q, v,
		"endpoint",
	)
}

func (v *RepositoryStorageS3) UpdateRegion(q *querier.Querier, region *string) error {
	v.Region = region
	return querier.UpdateOneFromStruct(q, v,
		"region",
	)
}

func (v *RepositoryStorageS3) UpdateBucket(q *querier.Querier, bucket string) error {
	v.Bucket = querier.N(bucket)
	return querier.UpdateOneFromStruct(q, v,
		"bucket",
	)
}

func (v *RepositoryStorageS3) UpdatePrefix(q *querier.Querier, prefix string) error {
	v.Prefix = querier.N(prefix)
	return querier.UpdateOneFromStruct(q, v,
		"prefix",
	)
}

func (v *RepositoryStorageS3) UpdateKeys(q *querier.Querier, accessKeyId string, secretAccessKey string) error {
	v.AccessKeyId = querier.N(accessKeyId)
	v.SecretAccessKey = querier.N(secretAccessKey)
	return querier.UpdateOneFromStruct(q, v,
		"access_key_id",
		"secret_access_key",
	)
}

func (v *RepositoryBackupRustic) UpdatePassword(q *querier.Querier, password string) error {
	v.Password = querier.N(password)
	return querier.UpdateOneFromStruct(q, v,
		"password",
	)
}
