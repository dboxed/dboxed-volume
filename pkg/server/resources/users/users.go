package users

import (
	"context"
	"slices"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dboxed/dboxed-common/db/querier"
	"github.com/dboxed/dboxed-common/huma_utils"
	"github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
	"github.com/dboxed/dboxed-volume/pkg/server/huma_metadata"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
)

type Users struct {
}

func New() *Users {
	return &Users{}
}

func (s *Users) Init(api huma.API) error {
	huma.Get(api, "/v1/admin/users", s.restListUsers, huma_metadata.NeedAdminModifier())
	huma.Get(api, "/v1/admin/users/{id}", s.restGetUser, huma_metadata.NeedAdminModifier())

	return nil
}

func (s *Users) restListUsers(ctx context.Context, i *struct{}) (*huma_utils.List[models.User], error) {
	q := querier.GetQuerier(ctx)
	config := config.GetConfig(ctx)

	l, err := dmodel.ListAllUsers(q)
	if err != nil {
		return nil, err
	}

	var ret []models.User
	for _, u := range l {
		isAdmin := slices.Contains(config.Auth.AdminUsers, u.ID)
		ret = append(ret, models.UserFromDB(u, isAdmin))
	}
	return huma_utils.NewList(ret, len(ret)), nil
}

func (s *Users) restGetUser(ctx context.Context, i *huma_utils.StringIdByPath) (*huma_utils.JsonBody[models.User], error) {
	q := querier.GetQuerier(ctx)
	config := config.GetConfig(ctx)

	v, err := dmodel.GetUserById(q, i.Id)
	if err != nil {
		return nil, err
	}
	isAdmin := slices.Contains(config.Auth.AdminUsers, v.ID)
	m := models.UserFromDB(*v, isAdmin)
	return huma_utils.NewJsonBody(m), nil
}
