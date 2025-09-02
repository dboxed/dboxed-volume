package s3proxy

import (
	"context"
	"log/slog"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dboxed/dboxed-common/huma_utils"
	"github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/repositories"
	"github.com/dboxed/dboxed-volume/pkg/server/s3utils"
	"github.com/minio/minio-go/v7"
)

type S3Proxy struct {
	bucketLocationCache sync.Map
}

type bucketLocationCacheKey struct {
	endpoint string
	bucket   string
}

func New(config config.Config) *S3Proxy {
	s := &S3Proxy{}
	return s
}

func (s *S3Proxy) Init(api huma.API) error {
	repoGroup := huma.NewGroup(api, "/v1/repositories/{repositoryId}")
	repoGroup.UseMiddleware(repositories.RepositoryMiddleware(api))

	huma.Post(repoGroup, "/s3proxy/list-objects", s.restListObjects)
	huma.Post(repoGroup, "/s3proxy/presign-put", s.restPresignPut)
	huma.Post(repoGroup, "/s3proxy/rename-object", s.restRenameObject)
	huma.Post(repoGroup, "/s3proxy/delete-object", s.restDeleteObject)

	return nil
}

func (s *S3Proxy) handleBase(ctx context.Context) (*dmodel.Repository, *minio.Client, error) {
	r := repositories.GetRepository(ctx)

	region := r.S3.Region

	if region == nil {
		key := bucketLocationCacheKey{
			endpoint: r.S3.Endpoint.V,
			bucket:   r.S3.Bucket.V,
		}

		cachedRegion, ok := s.bucketLocationCache.Load(key)
		if !ok {
			c, err := s3utils.BuildS3ClientForRegion(r, "")
			if err != nil {
				return nil, nil, err
			}
			loc, err := c.GetBucketLocation(ctx, r.S3.Bucket.V)
			if err != nil {
				return nil, nil, err
			}
			cachedRegion = &loc
			s.bucketLocationCache.Store(key, cachedRegion)
		}
		region = cachedRegion.(*string)
	}

	c, err := s3utils.BuildS3ClientForRegion(r, *region)
	if err != nil {
		return nil, nil, err
	}
	return r, c, nil
}

func (s *S3Proxy) restListObjects(ctx context.Context, i *huma_utils.JsonBody[models.S3ProxyListObjectsRequest]) (*huma_utils.JsonBody[models.S3ProxyListObjectsResult], error) {
	r, c, err := s.handleBase(ctx)
	if err != nil {
		return nil, err
	}

	slog.InfoContext(ctx, "restListObjects", slog.Any("repoPrefix", r.S3.Prefix.V), slog.Any("listPrefix", i.Body.Prefix))

	prefix := path.Join(r.S3.Prefix.V, i.Body.Prefix)
	if strings.HasSuffix(i.Body.Prefix, "/") {
		prefix += "/"
	}

	ch := c.ListObjects(ctx, r.S3.Bucket.V, minio.ListObjectsOptions{
		Prefix: prefix,
	})

	rep := models.S3ProxyListObjectsResult{}
	for o := range ch {
		presignedGetUrl, expires, err := s.presignGet(ctx, r, c, o.Key)
		if err != nil {
			return nil, err
		}
		oi := models.S3ObjectInfo{
			Key:          strings.TrimPrefix(o.Key, r.S3.Prefix.V+"/"),
			Size:         o.Size,
			LastModified: &o.LastModified,
			Etag:         o.ETag,

			PresignedGetUrl:        presignedGetUrl,
			PresignedGetUrlExpires: expires,
		}
		rep.Objects = append(rep.Objects, oi)
	}

	return huma_utils.NewJsonBody(rep), nil
}

func (s *S3Proxy) presignGet(ctx context.Context, r *dmodel.Repository, c *minio.Client, key string) (string, time.Time, error) {
	expiry := time.Hour
	expires := time.Now().Add(expiry).Add(time.Second * 15)
	pr, err := c.PresignedGetObject(ctx, r.S3.Bucket.V, key, expiry, nil)
	if err != nil {
		return "", time.Time{}, err
	}

	return pr.String(), expires, nil
}

func (s *S3Proxy) restPresignPut(ctx context.Context, i *huma_utils.JsonBody[models.S3ProxyPresignPutRequest]) (*huma_utils.JsonBody[models.S3ProxyPresignPutResult], error) {
	r, c, err := s.handleBase(ctx)
	if err != nil {
		return nil, err
	}

	key := path.Join(r.S3.Prefix.V, i.Body.Key)

	expiry := time.Hour
	expires := time.Now().Add(expiry).Add(time.Second * 15)
	pr, err := c.PresignedPutObject(ctx, r.S3.Bucket.V, key, expiry)
	if err != nil {
		return nil, err
	}

	return huma_utils.NewJsonBody(models.S3ProxyPresignPutResult{
		PresignedUrl: pr.String(),
		Expires:      expires,
	}), nil
}

func (s *S3Proxy) restRenameObject(ctx context.Context, i *huma_utils.JsonBody[models.S3ProxyRenameObjectRequest]) (*huma_utils.JsonBody[models.S3ProxyRenameObjectResult], error) {
	r, c, err := s.handleBase(ctx)
	if err != nil {
		return nil, err
	}

	oldKey := path.Join(r.S3.Prefix.V, i.Body.OldKey)
	newKey := path.Join(r.S3.Prefix.V, i.Body.NewKey)

	_, err = c.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: r.S3.Bucket.V,
		Object: newKey,
	}, minio.CopySrcOptions{
		Bucket: r.S3.Bucket.V,
		Object: oldKey,
	})
	if err != nil {
		return nil, err
	}
	err = c.RemoveObject(ctx, r.S3.Bucket.V, oldKey, minio.RemoveObjectOptions{})
	if err != nil {
		return nil, err
	}
	rep := models.S3ProxyRenameObjectResult{}
	return huma_utils.NewJsonBody(rep), nil
}

func (s *S3Proxy) restDeleteObject(ctx context.Context, i *huma_utils.JsonBody[models.S3ProxyDeleteObjectRequest]) (*huma_utils.JsonBody[models.S3ProxyDeleteObjectResult], error) {
	r, c, err := s.handleBase(ctx)
	if err != nil {
		return nil, err
	}

	key := path.Join(r.S3.Prefix.V, i.Body.Key)

	err = c.RemoveObject(ctx, r.S3.Bucket.V, key, minio.RemoveObjectOptions{})
	if err != nil {
		return nil, err
	}
	rep := models.S3ProxyDeleteObjectResult{}
	return huma_utils.NewJsonBody(rep), nil
}
