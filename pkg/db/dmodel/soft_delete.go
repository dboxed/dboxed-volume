package dmodel

import (
	"github.com/dboxed/dboxed-common/db/querier"
	"github.com/dboxed/dboxed-common/db/soft_delete"
)

func SoftDeleteWithConstraintsByIds[T querier.HasId](q *querier.Querier, id int64) error {
	err := soft_delete.SoftDeleteWithConstraints[T](q, map[string]any{
		"id": id,
	})
	if err != nil {
		return err
	}
	return nil
}
