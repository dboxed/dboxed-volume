package dmodel

import (
	"time"

	"github.com/dboxed/dboxed-common/db/querier"
	"github.com/dboxed/dboxed-common/util"
)

type Volume struct {
	Base

	Name string `db:"name"`
	Uuid string `db:"uuid"`

	RepositoryID int64 `db:"repository_id"`

	FsSize int64  `db:"fs_size"`
	FsType string `db:"fs_type"`

	LockId   *string `db:"lock_id"`
	LockTime *int64  `db:"lock_time"`

	Repository *Repository `join:"true"`
}

func (v *Volume) Create(q *querier.Querier) error {
	return querier.Create(q, v)
}

func GetVolumeById(q *querier.Querier, repositoryId *int64, id int64, skipDeleted bool) (*Volume, error) {
	return querier.GetOne[Volume](q, map[string]any{
		"repository_id": querier.OmitIfNull(repositoryId),
		"id":            id,
		"deleted_at":    querier.ExcludeNonNull(skipDeleted),
	})
}

func GetVolumeByUuid(q *querier.Querier, uuid string, skipDeleted bool) (*Volume, error) {
	return querier.GetOne[Volume](q, map[string]any{
		"uuid":       uuid,
		"deleted_at": querier.ExcludeNonNull(skipDeleted),
	})
}

func GetVolumeByName(q *querier.Querier, repositoryId int64, name string, skipDeleted bool) (*Volume, error) {
	return querier.GetOne[Volume](q, map[string]any{
		"repository_id": repositoryId,
		"name":          name,
		"deleted_at":    querier.ExcludeNonNull(skipDeleted),
	})
}

func ListVolumesForRepository(q *querier.Querier, repositoryId int64, skipDeleted bool) ([]Volume, error) {
	return querier.GetMany[Volume](q, map[string]any{
		"repository_id": repositoryId,
		"deleted_at":    querier.ExcludeNonNull(skipDeleted),
	})
}

func (v *Volume) UpdateLock(q *querier.Querier, newLockId string, newLockTime time.Time) error {
	oldLockId := v.LockId
	oldLockTime := v.LockTime
	v.LockId = &newLockId
	v.LockTime = util.Ptr(newLockTime.Unix())
	return querier.UpdateOneByFields[Volume](q, map[string]any{
		"id":        v.ID,
		"lock_id":   oldLockId,
		"lock_time": oldLockTime,
	}, map[string]any{
		"lock_id":   v.LockId,
		"lock_time": v.LockTime,
	})
}
