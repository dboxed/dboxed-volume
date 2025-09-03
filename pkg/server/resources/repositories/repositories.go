package repositories

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strconv"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dboxed/dboxed-common/db/querier"
	"github.com/dboxed/dboxed-common/huma_utils"
	"github.com/dboxed/dboxed-common/util"
	"github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
	"github.com/dboxed/dboxed-volume/pkg/server/huma_metadata"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/auth"
	"github.com/google/uuid"
)

type Repositories struct {
}

func New(config config.Config) *Repositories {
	s := &Repositories{}
	return s
}

func (s *Repositories) Init(api huma.API) error {

	huma.Post(api, "/v1/repositories", s.restCreateRepository)
	huma.Get(api, "/v1/repositories", s.restListRepositories)
	huma.Get(api, "/v1/repositories/{repositoryId}", s.restGetRepository)
	huma.Get(api, "/v1/repositories/by-name/{repositoryName}", s.restGetRepositoryByName)
	huma.Patch(api, "/v1/repositories/{repositoryId}", s.restUpdateRepository)
	huma.Delete(api, "/v1/repositories/{repositoryId}", s.restDeleteRepository)

	huma.Get(api, "/v1/admin/repositories", s.restAdminListRepositories, huma_metadata.NeedAdminModifier())

	return nil
}

func (s *Repositories) restCreateRepository(ctx context.Context, i *huma_utils.JsonBody[models.CreateRepository]) (*huma_utils.JsonBody[models.Repository], error) {
	q := querier.GetQuerier(ctx)
	user := auth.MustGetUser(ctx)

	err := util.CheckName(i.Body.Name)
	if err != nil {
		return nil, huma.Error400BadRequest("invalid name", err)
	}

	if i.Body.S3 == nil {
		return nil, huma.Error400BadRequest("currently only S3 storage is supported")
	}
	if i.Body.Rustic == nil {
		return nil, huma.Error400BadRequest("currently only rustic is supported")
	}

	if i.Body.Rustic != nil {
		if i.Body.Rustic.Password == "" {
			return nil, huma.Error400BadRequest("rustic password is missing")
		}
	}

	r := dmodel.Repository{
		Uuid: uuid.NewString(),
		Name: i.Body.Name,
	}

	err = r.Create(q)
	if err != nil {
		return nil, err
	}

	ra := dmodel.RepositoryAccess{
		RepositoryId: r.ID,
		UserId:       user.ID,
	}
	err = ra.Create(q)
	if err != nil {
		return nil, err
	}

	if i.Body.S3 != nil {
		err = s.checkEndpoint(i.Body.S3.Endpoint)
		if err != nil {
			return nil, err
		}
		if i.Body.S3.Prefix != "" {
			err = s.checkPrefix(i.Body.S3.Prefix)
			if err != nil {
				return nil, err
			}
		}

		r.S3 = &dmodel.RepositoryStorageS3{
			ID:              querier.N(r.ID),
			Endpoint:        querier.N(i.Body.S3.Endpoint),
			Region:          i.Body.S3.Region,
			Bucket:          querier.N(i.Body.S3.Bucket),
			AccessKeyId:     querier.N(i.Body.S3.AccessKeyId),
			SecretAccessKey: querier.N(i.Body.S3.SecretAccessKey),
			Prefix:          querier.N(i.Body.S3.Prefix),
		}
		err = r.S3.Create(q)
		if err != nil {
			return nil, err
		}
	}
	if i.Body.Rustic != nil {
		r.Rustic = &dmodel.RepositoryBackupRustic{
			ID:       querier.N(r.ID),
			Password: querier.N(i.Body.Rustic.Password),
		}
		err = r.Rustic.Create(q)
		if err != nil {
			return nil, err
		}
	}

	return huma_utils.NewJsonBody(models.RepositoryFromDB(r)), nil
}

func (s *Repositories) restListRepositories(ctx context.Context, i *struct{}) (*huma_utils.List[models.Repository], error) {
	return s.doRestListRepositories(ctx, i, false)
}

func (s *Repositories) restAdminListRepositories(ctx context.Context, i *struct{}) (*huma_utils.List[models.Repository], error) {
	return s.doRestListRepositories(ctx, i, true)
}

func (s *Repositories) doRestListRepositories(ctx context.Context, i *struct{}, asAdmin bool) (*huma_utils.List[models.Repository], error) {
	q := querier.GetQuerier(ctx)
	user := auth.MustGetUser(ctx)

	var l []dmodel.Repository
	var err error
	if asAdmin {
		l, err = dmodel.ListRepositories(q, nil, true)
	} else {
		l, err = dmodel.ListRepositories(q, &user.ID, true)
	}
	if err != nil {
		return nil, err
	}

	var ret []models.Repository
	for _, r := range l {
		mm := models.RepositoryFromDB(r)
		ret = append(ret, mm)
	}
	return huma_utils.NewList(ret, len(ret)), nil
}

type RepositoryId struct {
	RepositoryId int64 `path:"repositoryId"`
}

func (s *Repositories) restGetRepository(c context.Context, i *RepositoryId) (*huma_utils.JsonBody[models.Repository], error) {
	r, err := checkRepositoryAccess(c, i.RepositoryId)
	if err != nil {
		return nil, err
	}

	m := models.RepositoryFromDB(*r)
	return huma_utils.NewJsonBody(m), nil
}

type RepositoryName struct {
	RepositoryName string `path:"repositoryName"`
}

func (s *Repositories) restGetRepositoryByName(c context.Context, i *RepositoryName) (*huma_utils.JsonBody[models.Repository], error) {
	q := querier.GetQuerier(c)

	r, err := dmodel.GetRepositoryByName(q, i.RepositoryName, true)
	if err != nil {
		return nil, err
	}
	_, err = checkRepositoryAccess(c, r.ID)
	if err != nil {
		return nil, err
	}

	m := models.RepositoryFromDB(*r)
	return huma_utils.NewJsonBody(m), nil
}

type restUpdateRepositoryInput struct {
	RepositoryId
	huma_utils.JsonBody[models.UpdateRepository]
}

func (s *Repositories) restUpdateRepository(c context.Context, i *restUpdateRepositoryInput) (*huma_utils.JsonBody[models.Repository], error) {
	r, err := checkRepositoryAccess(c, i.RepositoryId.RepositoryId)
	if err != nil {
		return nil, err
	}

	err = s.doUpdateRepository(c, r, i.Body)
	if err != nil {
		return nil, err
	}

	m := models.RepositoryFromDB(*r)

	return huma_utils.NewJsonBody(m), nil
}

func (s *Repositories) doUpdateRepository(c context.Context, r *dmodel.Repository, body models.UpdateRepository) error {
	q := querier.GetQuerier(c)
	if body.S3 != nil {
		if body.S3.Endpoint != nil {
			err := s.checkEndpoint(*body.S3.Endpoint)
			if err != nil {
				return err
			}
			err = r.S3.UpdateEndpoint(q, *body.S3.Endpoint)
			if err != nil {
				return err
			}
		}
		if body.S3.Region != nil {
			err := r.S3.UpdateRegion(q, body.S3.Region)
			if err != nil {
				return err
			}
		}
		if body.S3.Bucket != nil {
			err := r.S3.UpdateBucket(q, *body.S3.Bucket)
			if err != nil {
				return err
			}
		}
		if body.S3.Prefix != nil {
			err := s.checkPrefix(*body.S3.Prefix)
			if err != nil {
				return err
			}
			err = r.S3.UpdatePrefix(q, *body.S3.Prefix)
			if err != nil {
				return err
			}
		}

		if body.S3.AccessKeyId != nil || body.S3.SecretAccessKey != nil {
			if body.S3.AccessKeyId == nil || body.S3.SecretAccessKey == nil {
				return huma.Error400BadRequest("either all or none of accessKeyId and secretAccessKey must be set")
			}
			err := r.S3.UpdateKeys(q, *body.S3.AccessKeyId, *body.S3.SecretAccessKey)
			if err != nil {
				return err
			}
		}
	}
	if body.Rustic != nil {
		if body.Rustic.Password != nil {
			if *body.Rustic.Password == "" {
				return huma.Error400BadRequest("rustic password can not be empty")
			}
			err := r.Rustic.UpdatePassword(q, *body.Rustic.Password)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Repositories) restDeleteRepository(c context.Context, i *RepositoryId) (*huma_utils.Empty, error) {
	q := querier.GetQuerier(c)

	_, err := checkRepositoryAccess(c, i.RepositoryId)
	if err != nil {
		return nil, err
	}

	err = dmodel.SoftDeleteWithConstraintsByIds[dmodel.Repository](q, i.RepositoryId)
	if err != nil {
		return nil, err
	}

	return &huma_utils.Empty{}, nil
}

func checkRepositoryAccess(ctx context.Context, id int64) (*dmodel.Repository, error) {
	q := querier.GetQuerier(ctx)
	user := auth.MustGetUser(ctx)

	r, err := dmodel.GetRepositoryById(q, id, true)
	if err != nil {
		if util.IsSqlNotFoundError(err) {
			return nil, huma.Error404NotFound("workspace not found")
		}
		return nil, err
	}

	if !user.IsAdmin {
		if !slices.ContainsFunc(r.Access, func(access dmodel.RepositoryAccess) bool {
			return access.UserId == user.ID
		}) {
			return nil, huma.Error403Forbidden("access to repository not allowed")
		}
	}

	return r, nil
}

func RepositoryMiddleware(api huma.API) func(ctx huma.Context, next func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		repositoryIdStr := ctx.Param("repositoryId")
		if repositoryIdStr == "" {
			huma.WriteErr(api, ctx, http.StatusBadRequest, "missing workspace id")
			return
		}

		repositoryId, err := strconv.ParseInt(repositoryIdStr, 10, 64)
		if err != nil {
			huma.WriteErr(api, ctx, http.StatusBadRequest, "invalid workspace id", err)
			return
		}

		r, err := checkRepositoryAccess(ctx.Context(), repositoryId)
		if err != nil {
			var err2 huma.StatusError
			if errors.As(err, &err2) {
				huma.WriteErr(api, ctx, err2.GetStatus(), err.Error(), err)
			} else {
				huma.WriteErr(api, ctx, http.StatusForbidden, err.Error(), err)
			}
			return
		}

		ctx = huma.WithValue(ctx, "repository", r)

		next(ctx)
	}
}

func GetRepository(ctx context.Context) *dmodel.Repository {
	i := ctx.Value("repository")
	if i == nil {
		panic("no repository in context")
	}
	return i.(*dmodel.Repository)
}

func (s *Repositories) checkEndpoint(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return huma.Error400BadRequest("invalid endpoint", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return huma.Error400BadRequest("invalid endpoint scheme")
	}
	return nil
}

var prefixRegex = regexp.MustCompile(`^([a-zA-Z0-9]*)(/([a-zA-Z0-9]+))*/?$`)

func (s *Repositories) checkPrefix(prefix string) error {
	if !prefixRegex.MatchString(prefix) {
		return fmt.Errorf("invalid prefix")
	}
	return nil
}
