package models

import (
	"time"

	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
)

type Volume struct {
	ID        int64     `json:"id"`
	CreatedAt time.Time `json:"createdAt"`

	Name string `json:"name"`
	Uuid string `json:"uuid"`

	RepositoryID int64 `json:"repositoryId"`

	FsSize int64  `json:"fsSize"`
	FsType string `json:"fsType"`

	LockId   *string `json:"lockId,omitempty"`
	LockTime *int64  `json:"lockTime,omitempty"`
}

type CreateVolume struct {
	Name   string `json:"name"`
	FsSize int64  `json:"fsSize"`
	FsType string `json:"fsType"`
}

type VolumeLockRequest struct {
	PrevLockId *string `json:"prevLockId"`
}

func VolumeFromDB(v dmodel.Volume) Volume {
	ret := Volume{
		ID:           v.ID,
		CreatedAt:    v.CreatedAt,
		Name:         v.Name,
		Uuid:         v.Uuid,
		RepositoryID: v.RepositoryID,
		FsSize:       v.FsSize,
		FsType:       v.FsType,
		LockId:       v.LockId,
		LockTime:     v.LockTime,
	}
	return ret
}
