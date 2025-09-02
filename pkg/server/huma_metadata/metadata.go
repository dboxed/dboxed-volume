package huma_metadata

import (
	"github.com/danielgtaylor/huma/v2"
	"github.com/dboxed/dboxed-common/huma_utils"
)

const SkipAuth = "skip-auth"
const NeedAdmin = "need-admin"

func NeedAdminModifier() func(o *huma.Operation) {
	return huma_utils.MetadataModifier(NeedAdmin, true)
}
