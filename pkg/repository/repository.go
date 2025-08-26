package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dboxed/dboxed-volume/pkg/util"
	"github.com/dboxed/dboxed-volume/pkg/volume"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
)

type RepositoryStore struct {
	nc      *nats.Conn
	js      nats.JetStreamContext
	reposKv nats.KeyValue

	replicas int
}

type Repository struct {
	rs *RepositoryStore

	Repo *dproto.Repository
	rev  uint64

	volumesKv nats.KeyValue
}

type Volume struct {
	r *Repository

	Volume *dproto.Volume
	rev    uint64
}

func OpenStore(nc *nats.Conn, replicas int) (*RepositoryStore, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	repoBucket := "dboxed-volumes-repositories"
	kv, err := js.KeyValue(repoBucket)
	if err != nil {
		if errors.Is(err, nats.ErrBucketNotFound) {
			kv, err = js.CreateKeyValue(&nats.KeyValueConfig{
				Bucket:   repoBucket,
				Replicas: replicas,
			})
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}

	rs := &RepositoryStore{
		nc:       nc,
		js:       js,
		reposKv:  kv,
		replicas: replicas,
	}
	return rs, nil
}

func (r *Repository) queryRepo(uuid string) error {
	v, err := r.rs.reposKv.Get(buildUuidKey(uuid))
	if err != nil {
		return err
	}
	var x dproto.Repository
	err = json.Unmarshal(v.Value(), &x)
	if err != nil {
		return err
	}
	r.Repo = &x
	r.rev = v.Revision()
	return nil
}

func (r *Repository) writeRepo(create bool) error {
	j, err := json.Marshal(r.Repo)
	if err != nil {
		return err
	}

	uuidKey := buildUuidKey(r.Repo.Uuid)

	if create {
		rev, err := r.rs.reposKv.Create(uuidKey, j)
		if err != nil {
			return err
		}
		r.rev = rev

		_, err = r.rs.reposKv.Create(buildNameKey(r.Repo.Name), []byte(r.Repo.Uuid))
		if err != nil {
			_ = r.rs.reposKv.Delete(uuidKey)
			return err
		}
	} else {
		rev, err := r.rs.reposKv.Update(uuidKey, j, r.rev)
		if err != nil {
			return err
		}
		r.rev = rev
	}
	return nil
}

func buildNameKey(name string) string {
	return fmt.Sprintf("by-name.%s", name)
}

func buildUuidKey(uuid string) string {
	return fmt.Sprintf("by-uuid.%s", uuid)
}

func (rs *RepositoryStore) Create(ctx context.Context, req *dproto.RepositoryCreateRequest) (*Repository, error) {
	err := util.CheckName(req.Name)
	if err != nil {
		return nil, err
	}

	// ensure we don't create a conflicting repo
	_, err = rs.reposKv.Get(buildUuidKey(req.Name))
	if err != nil {
		if !errors.Is(err, nats.ErrKeyNotFound) {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("conflicting name")
	}

	slog.Info("creating repository", slog.Any("name", req.Name))

	r := &Repository{
		rs: rs,
		Repo: &dproto.Repository{
			Name: req.Name,
			Uuid: uuid.NewString(),
		},
	}

	if req.S3 != nil {
		r.Repo.S3 = req.S3
	} else {
		return nil, fmt.Errorf("missing backend settings")
	}

	err = r.queryBucketLocation(ctx)
	if err != nil {
		return nil, err
	}

	if req.S3 != nil {
		err = r.initS3Repo(ctx)
		if err != nil {
			return nil, err
		}
	}

	err = r.writeRepo(true)
	if err != nil {
		return nil, err
	}

	bucket := fmt.Sprintf("dboxed-volumes-%s", r.Repo.Uuid)
	r.volumesKv, err = rs.js.CreateKeyValue(&nats.KeyValueConfig{
		Bucket:   bucket,
		Replicas: rs.replicas,
	})
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (rs *RepositoryStore) OpenByUuid(uuid string) (*Repository, error) {
	r := &Repository{
		rs: rs,
	}

	err := r.queryRepo(uuid)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func (rs *RepositoryStore) OpenByName(name string) (*Repository, error) {
	key := buildNameKey(name)
	v, err := rs.reposKv.Get(key)
	if err != nil {
		return nil, err
	}
	return rs.OpenByUuid(string(v.Value()))
}

func (r *Repository) initVolumesKV() error {
	if r.volumesKv != nil {
		return nil
	}

	bucket := fmt.Sprintf("dboxed-volumes-%s", r.Repo.Uuid)
	volumesKv, err := r.rs.js.KeyValue(bucket)
	if err != nil {
		return err
	}
	r.volumesKv = volumesKv
	return nil
}

func (r *Repository) buildS3ClientForRegion(region string) (*s3.Client, error) {
	if r.Repo.S3 == nil {
		return nil, fmt.Errorf("not a S3 repository")
	}

	creds := credentials.NewStaticCredentialsProvider(r.Repo.S3.AccessKeyId, r.Repo.S3.SecretAccessKey, "")
	endpoint := fmt.Sprintf("https://%s", r.Repo.S3.Endpoint)

	cfg := aws.Config{
		Region:       region,
		Credentials:  creds,
		BaseEndpoint: &endpoint,
	}

	client := s3.NewFromConfig(cfg)

	return client, nil
}

func (r *Repository) BuildS3Client() (*s3.Client, error) {
	if r.Repo.S3 == nil {
		return nil, fmt.Errorf("not a S3 repository")
	}

	region := ""
	if r.Repo.S3.Region != nil {
		region = *r.Repo.S3.Region
	}

	return r.buildS3ClientForRegion(region)
}

func (r *Repository) queryBucketLocation(ctx context.Context) error {
	if r.Repo.S3.Region != nil {
		return nil
	}

	mc, err := r.buildS3ClientForRegion("")
	if err != nil {
		return err
	}
	location, err := mc.GetBucketLocation(ctx, &s3.GetBucketLocationInput{
		Bucket: &r.Repo.S3.Bucket,
	})
	if err != nil {
		return err
	}
	r.Repo.S3.Region = util.Ptr(string(location.LocationConstraint))
	return nil
}

func (r *Repository) initS3Repo(ctx context.Context) error {
	return nil
}

func (r *Repository) Update(ctx context.Context, req *dproto.RepositoryUpdateRequest) error {
	if req.S3 != nil {
		r.Repo.S3 = req.S3
	} else {
		return fmt.Errorf("missing backend settings")
	}

	r.Repo.S3 = req.S3

	err := r.queryBucketLocation(ctx)
	if err != nil {
		return err
	}

	err = r.writeRepo(false)
	if err != nil {
		return err
	}
	return nil
}

func (r *Repository) CreateVolume(name string, fsSize int64, fsType string) (*Volume, error) {
	err := util.CheckName(name)
	if err != nil {
		return nil, err
	}
	if fsSize <= 0 {
		return nil, fmt.Errorf("invalid fs_size")
	}
	if !slices.Contains(volume.AllowedFsTypes, fsType) {
		return nil, fmt.Errorf("invalid fs_type")
	}

	err = r.initVolumesKV()
	if err != nil {
		return nil, err
	}

	// ensure we don't create a conflicting repo
	_, err = r.volumesKv.Get(buildUuidKey(name))
	if err != nil {
		if !errors.Is(err, nats.ErrKeyNotFound) {
			return nil, err
		}
	} else {
		return nil, fmt.Errorf("conflicting name")
	}

	slog.Info("creating volume", slog.Any("repositoryUuid", r.Repo.Uuid), slog.Any("volumeName", name))

	v := &dproto.Volume{
		Uuid:           uuid.NewString(),
		RepositoryUuid: r.Repo.Uuid,
		Name:           name,
		FsSize:         fsSize,
		FsType:         fsType,
	}
	j, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}

	rev, err := r.volumesKv.Create(buildUuidKey(v.Uuid), j)
	if err != nil {
		return nil, err
	}
	_, err = r.volumesKv.Create(buildNameKey(name), []byte(v.Uuid))
	if err != nil {
		_ = r.volumesKv.Delete(buildUuidKey(v.Uuid))
		return nil, err
	}

	return &Volume{
		r:      r,
		Volume: v,
		rev:    rev,
	}, nil
}

func (r *Repository) OpenVolumeByUuid(uuid string) (*Volume, error) {
	v := &Volume{
		r: r,
	}

	err := v.queryVolume(uuid)
	if err != nil {
		return nil, err
	}
	return v, nil
}

func (r *Repository) OpenVolumeByName(name string) (*Volume, error) {
	err := r.initVolumesKV()
	if err != nil {
		return nil, err
	}

	key := buildNameKey(name)
	v, err := r.volumesKv.Get(key)
	if err != nil {
		return nil, err
	}
	return r.OpenVolumeByUuid(string(v.Value()))
}

func (r *Volume) Update() error {
	return r.writeVolume(false)
}

func (r *Volume) queryVolume(uuid string) error {
	err := r.r.initVolumesKV()
	if err != nil {
		return err
	}

	v, err := r.r.volumesKv.Get(buildUuidKey(uuid))
	if err != nil {
		return err
	}
	var x dproto.Volume
	err = json.Unmarshal(v.Value(), &x)
	if err != nil {
		return err
	}
	r.Volume = &x
	r.rev = v.Revision()
	return nil
}

func (r *Volume) writeVolume(create bool) error {
	err := r.r.initVolumesKV()
	if err != nil {
		return err
	}

	j, err := json.Marshal(r.Volume)
	if err != nil {
		return err
	}

	uuidKey := buildUuidKey(r.Volume.Uuid)

	if create {
		rev, err := r.r.volumesKv.Create(uuidKey, j)
		if err != nil {
			return err
		}
		r.rev = rev

		_, err = r.r.volumesKv.Create(buildNameKey(r.Volume.Name), []byte(r.Volume.Uuid))
		if err != nil {
			_ = r.r.volumesKv.Delete(uuidKey)
			return err
		}
	} else {
		rev, err := r.r.volumesKv.Update(uuidKey, j, r.rev)
		if err != nil {
			return err
		}
		r.rev = rev
	}
	return nil
}
