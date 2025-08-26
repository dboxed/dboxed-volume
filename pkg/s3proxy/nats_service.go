package s3proxy

import (
	"bytes"
	"context"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dboxed/dboxed-volume/pkg/nats/nats_handlers"
	"github.com/dboxed/dboxed-volume/pkg/repository"
	"github.com/dboxed/dboxed-volume/pkg/util"
	"github.com/dboxed/dboxed-volume/pkg/version"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const NatsServiceName = "dboxed-volume-s3-proxy"

type NatsS3ProxyService struct {
	ctx context.Context
	nc  *nats.Conn
	rs  *repository.RepositoryStore
}

func NewNatsService(ctx context.Context, nc *nats.Conn, rs *repository.RepositoryStore) (*NatsS3ProxyService, error) {
	s := &NatsS3ProxyService{
		ctx: ctx,
		nc:  nc,
		rs:  rs,
	}
	return s, nil
}

func (s *NatsS3ProxyService) Start() error {
	srv, err := micro.AddService(s.nc, micro.Config{
		Name:    NatsServiceName,
		Version: strings.TrimPrefix(version.Version, "v"),
	})
	if err != nil {
		return err
	}

	g := srv.AddGroup(NatsServiceName)
	sem := semaphore.NewWeighted(8)

	err = g.AddEndpoint("small-put", nats_handlers.BuildHandler(s.ctx, sem, s.handleSmallPut))
	if err != nil {
		return err
	}

	err = g.AddEndpoint("presign-get", nats_handlers.BuildHandler(s.ctx, sem, s.handlePresignGet))
	if err != nil {
		return err
	}
	err = g.AddEndpoint("presign-put", nats_handlers.BuildHandler(s.ctx, sem, s.handlePresignPut))
	if err != nil {
		return err
	}

	err = g.AddEndpoint("list-objects", nats_handlers.BuildHandler(s.ctx, sem, s.handleListObjects))
	if err != nil {
		return err
	}
	err = g.AddEndpoint("rename-object", nats_handlers.BuildHandler(s.ctx, sem, s.handleRenameObject))
	if err != nil {
		return err
	}
	err = g.AddEndpoint("remove-object", nats_handlers.BuildHandler(s.ctx, sem, s.handleRemoveObject))
	if err != nil {
		return err
	}

	err = g.AddEndpoint("create-multipart-upload", nats_handlers.BuildHandler(s.ctx, sem, s.handleCreateMultipartUpload))
	if err != nil {
		return err
	}
	err = g.AddEndpoint("presign-multipart-upload", nats_handlers.BuildHandler(s.ctx, sem, s.handlePresignMultipartUpload))
	if err != nil {
		return err
	}
	err = g.AddEndpoint("complete-multipart-upload", nats_handlers.BuildHandler(s.ctx, sem, s.handleCompleteMultipartUpload))
	if err != nil {
		return err
	}
	return nil
}

func (s *NatsS3ProxyService) handleBase(repoUuid string) (*repository.Repository, *s3.Client, error) {

	r, err := s.rs.OpenByUuid(repoUuid)
	if err != nil {
		return nil, nil, err
	}

	mc, err := r.BuildS3Client()
	if err != nil {
		return nil, nil, err
	}

	return r, mc, nil
}

func (s *NatsS3ProxyService) handleSmallPut(ctx context.Context, req *dproto.S3ProxySmallPutRequest) (*dproto.S3ProxySmallPutReply, error) {
	defer util.Measure("handleSmallPut")()

	if len(req.Body) > 512 {
		return nil, fmt.Errorf("body too large")
	}

	r, mc, err := s.handleBase(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	objectName := path.Join(r.Repo.S3.Prefix, req.ObjectName)

	_, err = mc.PutObject(ctx, &s3.PutObjectInput{
		Bucket: &r.Repo.S3.Bucket,
		Key:    &objectName,
		Body:   bytes.NewReader(req.Body),
	})
	rep := &dproto.S3ProxySmallPutReply{}
	return rep, nil
}

func (s *NatsS3ProxyService) handlePresignGet(ctx context.Context, req *dproto.S3ProxyPresignGetRequest) (*dproto.S3ProxyPresignGetReply, error) {
	r, mc, err := s.handleBase(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	objectName := path.Join(r.Repo.S3.Prefix, req.ObjectName)

	pc := s3.NewPresignClient(mc)
	expiry := time.Hour
	pr, err := pc.PresignGetObject(ctx, &s3.GetObjectInput{
		Bucket: &r.Repo.S3.Bucket,
		Key:    &objectName,
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return nil, err
	}

	rep := &dproto.S3ProxyPresignGetReply{
		Url:     pr.URL,
		Expires: timestamppb.New(time.Now().Add(expiry).Add(-15 * time.Second)),
	}
	return rep, nil
}

func (s *NatsS3ProxyService) handlePresignPut(ctx context.Context, req *dproto.S3ProxyPresignPutRequest) (*dproto.S3ProxyPresignPutReply, error) {
	r, mc, err := s.handleBase(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	objectName := path.Join(r.Repo.S3.Prefix, req.ObjectName)

	pc := s3.NewPresignClient(mc)
	expiry := time.Hour
	pr, err := pc.PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &r.Repo.S3.Bucket,
		Key:    &objectName,
	}, s3.WithPresignExpires(expiry))
	if err != nil {
		return nil, err
	}

	rep := &dproto.S3ProxyPresignPutReply{
		Url:     pr.URL,
		Expires: timestamppb.New(time.Now().Add(expiry).Add(-15 * time.Second)),
	}
	return rep, nil
}

func (s *NatsS3ProxyService) handleListObjects(ctx context.Context, req *dproto.S3ProxyListObjectsRequest) (*dproto.S3ProxyListObjectsReply, error) {
	r, mc, err := s.handleBase(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	prefix := path.Join(r.Repo.S3.Prefix, req.Prefix)
	if strings.HasSuffix(req.Prefix, "/") {
		prefix += "/"
	}

	resp, err := mc.ListObjects(ctx, &s3.ListObjectsInput{
		Bucket:    &r.Repo.S3.Bucket,
		Prefix:    &prefix,
		Delimiter: util.Ptr("/"),
	})
	if err != nil {
		return nil, err
	}

	pc := s3.NewPresignClient(mc)

	rep := &dproto.S3ProxyListObjectsReply{}
	for _, cp := range resp.CommonPrefixes {
		trimmedPrefix := strings.TrimPrefix(*cp.Prefix, r.Repo.S3.Prefix+"/")
		rep.CommonPrefixes = append(rep.CommonPrefixes, trimmedPrefix)
	}
	for _, o := range resp.Contents {
		key := strings.TrimPrefix(*o.Key, r.Repo.S3.Prefix+"/")
		oi := &dproto.S3ObjectInfo{
			Key:          key,
			Size:         *o.Size,
			LastModified: timestamppb.New(*o.LastModified),
			Etag:         *o.ETag,
		}

		pr, err := pc.PresignGetObject(ctx, &s3.GetObjectInput{
			Bucket: &r.Repo.S3.Bucket,
			Key:    o.Key,
		}, s3.WithPresignExpires(time.Hour))
		if err != nil {
			return nil, err
		}
		oi.PresignedGetUrl = pr.URL

		rep.Objects = append(rep.Objects, oi)
	}
	return rep, nil
}

func (s *NatsS3ProxyService) handleRenameObject(ctx context.Context, req *dproto.S3ProxyRenameObjectRequest) (*dproto.S3ProxyRenameObjectReply, error) {
	r, mc, err := s.handleBase(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	oldObjectName := path.Join(r.Repo.S3.Prefix, req.OldObjectName)
	newObjectName := path.Join(r.Repo.S3.Prefix, req.NewObjectName)

	_, err = mc.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:     &r.Repo.S3.Bucket,
		CopySource: util.Ptr(fmt.Sprintf("%s/%s", r.Repo.S3.Bucket, oldObjectName)),
		Key:        &newObjectName,
	})
	if err != nil {
		return nil, err
	}
	_, err = mc.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &r.Repo.S3.Bucket,
		Key:    &oldObjectName,
	})
	if err != nil {
		return nil, err
	}
	return &dproto.S3ProxyRenameObjectReply{}, nil
}

func (s *NatsS3ProxyService) handleRemoveObject(ctx context.Context, req *dproto.S3ProxyRemoveObjectRequest) (*dproto.S3ProxyRemoveObjectReply, error) {
	r, mc, err := s.handleBase(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	objectName := path.Join(r.Repo.S3.Prefix, req.ObjectName)

	_, err = mc.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: &r.Repo.S3.Bucket,
		Key:    &objectName,
	})
	if err != nil {
		return nil, err
	}
	return &dproto.S3ProxyRemoveObjectReply{}, nil
}

func (s *NatsS3ProxyService) presignMultipartUploadParts(ctx context.Context, repositoryUuid string, objectName string, uploadId string, startPart int, count int) ([]string, error) {
	r, mc, err := s.handleBase(repositoryUuid)
	if err != nil {
		return nil, err
	}

	objectName = path.Join(r.Repo.S3.Prefix, objectName)

	var ret []string
	pc := s3.NewPresignClient(mc)
	for i := range count {
		pr, err := pc.PresignUploadPart(ctx, &s3.UploadPartInput{
			Bucket:     &r.Repo.S3.Bucket,
			Key:        &objectName,
			UploadId:   &uploadId,
			PartNumber: util.Ptr(int32(startPart + i)),
		})
		if err != nil {
			return nil, err
		}
		ret = append(ret, pr.URL)
	}
	return ret, nil
}

func (s *NatsS3ProxyService) handleCreateMultipartUpload(ctx context.Context, req *dproto.S3ProxyCreateMultipartUploadRequest) (*dproto.S3ProxyCreateMultipartUploadReply, error) {
	r, mc, err := s.handleBase(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	objectName := path.Join(r.Repo.S3.Prefix, req.ObjectName)

	expires := time.Now().Add(1 * time.Hour)
	resp, err := mc.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket:  &r.Repo.S3.Bucket,
		Key:     &objectName,
		Expires: &expires,
	})
	if err != nil {
		return nil, err
	}

	urls, err := s.presignMultipartUploadParts(ctx, req.RepositoryUuid, req.ObjectName, *resp.UploadId, 1, int(req.PresignedPartCount))
	if err != nil {
		return nil, err
	}

	rep := &dproto.S3ProxyCreateMultipartUploadReply{
		UploadId:            *resp.UploadId,
		PresignedUploadUrls: urls,
	}
	return rep, nil
}

func (s *NatsS3ProxyService) handlePresignMultipartUpload(ctx context.Context, req *dproto.S3ProxyPresignMultipartUploadRequest) (*dproto.S3ProxyPresignMultipartUploadReply, error) {
	urls, err := s.presignMultipartUploadParts(ctx, req.RepositoryUuid, req.ObjectName, req.UploadId, int(req.StartPart), int(req.Count))
	if err != nil {
		return nil, err
	}

	rep := &dproto.S3ProxyPresignMultipartUploadReply{
		Urls: urls,
	}

	return rep, nil
}

func (s *NatsS3ProxyService) handleCompleteMultipartUpload(ctx context.Context, req *dproto.S3ProxyCompleteMultipartUploadRequest) (*dproto.S3ProxyCompleteMultipartUploadReply, error) {
	r, mc, err := s.handleBase(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	objectName := path.Join(r.Repo.S3.Prefix, req.ObjectName)

	var completeParts []types.CompletedPart
	for _, cp := range req.CompleteParts {
		completeParts = append(completeParts, types.CompletedPart{
			PartNumber: &cp.PartNumer,
			ETag:       &cp.Etag,
		})
	}

	_, err = mc.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   &r.Repo.S3.Bucket,
		Key:      &objectName,
		UploadId: &req.UploadId,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completeParts,
		},
	})
	if err != nil {
		return nil, err
	}

	rep := &dproto.S3ProxyCompleteMultipartUploadReply{}
	return rep, nil
}
