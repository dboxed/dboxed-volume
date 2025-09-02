package dmodel

import (
	"time"

	"github.com/dboxed/dboxed-common/db/soft_delete"
)

type Times struct {
	CreatedAt time.Time `db:"created_at" omitCreate:"true"`
}

type Base struct {
	ID int64 `db:"id" omitCreate:"true"`
	Times
	soft_delete.SoftDeleteFields
}

func (v *Base) SetId(id int64) {
	v.ID = id
}

func (v Base) GetId() int64 {
	return v.ID
}
