package tokens

import (
	"context"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dboxed/dboxed-common/db/querier"
	"github.com/dboxed/dboxed-common/huma_utils"
	"github.com/dboxed/dboxed-common/util"
	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
	"github.com/dboxed/dboxed-volume/pkg/server/huma_metadata"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/auth"
	"github.com/google/uuid"
)

type Tokens struct {
}

func New() *Tokens {
	s := &Tokens{}
	return s
}

func (s *Tokens) Init(api huma.API) error {
	huma.Post(api, "/v1/tokens", s.restCreateToken, huma_metadata.NoTokenModifier())
	huma.Get(api, "/v1/tokens", s.restListTokens, huma_metadata.NoTokenModifier())
	huma.Get(api, "/v1/tokens/{id}", s.restGetToken, huma_metadata.NoTokenModifier())
	huma.Delete(api, "/v1/tokens/{id}", s.restDeleteToken, huma_metadata.NoTokenModifier())

	return nil
}

func (s *Tokens) restCreateToken(ctx context.Context, i *huma_utils.JsonBody[models.CreateToken]) (*huma_utils.JsonBody[models.CreateTokenResult], error) {
	q := querier.GetQuerier(ctx)
	user := auth.MustGetUser(ctx)

	err := util.CheckName(i.Body.Name)
	if err != nil {
		return nil, err
	}

	t := dmodel.Token{
		Token:  auth.TokenPrefix + uuid.NewString(),
		Name:   i.Body.Name,
		UserID: user.ID,
	}

	err = t.Create(q)
	if err != nil {
		return nil, err
	}

	return huma_utils.NewJsonBody(models.CreateTokenResult{
		Token:    models.TokenFromDB(t),
		TokenStr: t.Token,
	}), nil
}

func (s *Tokens) restListTokens(ctx context.Context, i *struct{}) (*huma_utils.List[models.Token], error) {
	q := querier.GetQuerier(ctx)
	user := auth.MustGetUser(ctx)

	l, err := dmodel.ListTokensForUser(q, user.ID)
	if err != nil {
		return nil, err
	}

	var ret []models.Token
	for _, r := range l {
		mm := models.TokenFromDB(r)
		ret = append(ret, mm)
	}
	return huma_utils.NewList(ret, len(ret)), nil
}

func (s *Tokens) restGetToken(c context.Context, i *huma_utils.IdByPath) (*huma_utils.JsonBody[models.Token], error) {
	q := querier.GetQuerier(c)
	user := auth.GetUser(c)

	t, err := dmodel.GetTokenById(q, &user.ID, i.Id)
	if err != nil {
		return nil, err
	}

	m := models.TokenFromDB(*t)
	return huma_utils.NewJsonBody(m), nil
}

func (s *Tokens) restDeleteToken(c context.Context, i *huma_utils.IdByPath) (*huma_utils.Empty, error) {
	q := querier.GetQuerier(c)
	user := auth.GetUser(c)

	err := querier.DeleteOneByFields[dmodel.Token](q, map[string]any{
		"id":      i.Id,
		"user_id": user.ID,
	})
	if err != nil {
		return nil, err
	}

	return &huma_utils.Empty{}, nil
}
