package server

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/dboxed/dboxed-volume/pkg/nats/dproto"
	"github.com/dboxed/dboxed-volume/pkg/nats/nats_handlers"
	"github.com/dboxed/dboxed-volume/pkg/repository"
	"github.com/dboxed/dboxed-volume/pkg/version"
	"github.com/google/uuid"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/micro"
	"golang.org/x/sync/semaphore"
	"google.golang.org/protobuf/types/known/timestamppb"
)

const lockTimeout = time.Minute

type Server struct {
	nc *nats.Conn
	js nats.JetStream

	rs *repository.RepositoryStore
}

func New(nc *nats.Conn, rs *repository.RepositoryStore) (*Server, error) {
	js, err := nc.JetStream()
	if err != nil {
		return nil, err
	}

	s := &Server{
		nc: nc,
		js: js,
		rs: rs,
	}

	return s, nil
}

func (s *Server) Start(ctx context.Context) error {
	fmt.Println("Starting dboxed-volume service")
	srv, err := micro.AddService(s.nc, micro.Config{
		Name:    "dboxed-volume",
		Version: strings.TrimPrefix(version.Version, "v"),
	})
	if err != nil {
		return err
	}

	rootGroup := srv.AddGroup("dboxed-volume")

	repoGroup := rootGroup.AddGroup("repository")
	sem := semaphore.NewWeighted(8)
	err = repoGroup.AddEndpoint("create", nats_handlers.BuildHandler(ctx, sem, s.handleRepositoryCreate))
	if err != nil {
		return err
	}
	err = repoGroup.AddEndpoint("update", nats_handlers.BuildHandler(ctx, sem, s.handleRepositoryUpdate))
	if err != nil {
		return err
	}
	err = repoGroup.AddEndpoint("by-name", nats_handlers.BuildHandler(ctx, sem, s.handleRepositoryByName))
	if err != nil {
		return err
	}

	volumeGroup := rootGroup.AddGroup("volume")
	err = volumeGroup.AddEndpoint("create", nats_handlers.BuildHandler(ctx, sem, s.handleVolumeCreate))
	if err != nil {
		return err
	}
	err = volumeGroup.AddEndpoint("by-name", nats_handlers.BuildHandler(ctx, sem, s.handleVolumeByName))
	if err != nil {
		return err
	}
	err = volumeGroup.AddEndpoint("lock", nats_handlers.BuildHandler(ctx, sem, s.handleVolumeLock))
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) handleRepositoryCreate(ctx context.Context, req *dproto.RepositoryCreateRequest) (*dproto.RepositoryCreateReply, error) {
	r, err := s.rs.Create(ctx, req)
	if err != nil {
		return nil, err
	}

	rep := &dproto.RepositoryCreateReply{
		Repository: r.Repo,
	}
	return rep, nil
}

func (s *Server) handleRepositoryUpdate(ctx context.Context, req *dproto.RepositoryUpdateRequest) (*dproto.RepositoryUpdateReply, error) {
	r, err := s.rs.OpenByUuid(req.Uuid)
	if err != nil {
		return nil, err
	}

	err = r.Update(ctx, req)
	if err != nil {
		return nil, err
	}

	rep := &dproto.RepositoryUpdateReply{
		Repository: r.Repo,
	}
	return rep, nil
}

func (s *Server) handleRepositoryByName(ctx context.Context, req *dproto.RepositoryByNameRequest) (*dproto.RepositoryByNameReply, error) {
	r, err := s.rs.OpenByName(req.Name)
	if err != nil {
		return nil, err
	}

	rep := &dproto.RepositoryByNameReply{
		Repository: r.Repo,
	}
	return rep, nil
}

func (s *Server) handleVolumeCreate(ctx context.Context, req *dproto.VolumeCreateRequest) (*dproto.VolumeCreateReply, error) {
	r, err := s.rs.OpenByUuid(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}

	v, err := r.CreateVolume(req.Name, req.FsSize, req.FsType)
	if err != nil {
		return nil, err
	}

	rep := &dproto.VolumeCreateReply{
		Volume: v.Volume,
	}
	return rep, nil
}

func (s *Server) handleVolumeByName(ctx context.Context, req *dproto.VolumeByNameRequest) (*dproto.VolumeByNameReply, error) {
	r, err := s.rs.OpenByUuid(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}
	v, err := r.OpenVolumeByName(req.Name)
	if err != nil {
		return nil, err
	}

	rep := &dproto.VolumeByNameReply{
		Volume: v.Volume,
	}
	return rep, nil
}

func (s *Server) handleVolumeLock(ctx context.Context, req *dproto.VolumeLockRequest) (*dproto.VolumeLockReply, error) {
	r, err := s.rs.OpenByUuid(req.RepositoryUuid)
	if err != nil {
		return nil, err
	}
	v, err := r.OpenVolumeByUuid(req.VolumeUuid)
	if err != nil {
		return nil, err
	}

	log := slog.With(
		slog.Any("repository", v.Volume.RepositoryUuid),
		slog.Any("uuid", v.Volume.Uuid),
		slog.Any("name", v.Volume.Name),
	)
	if v.Volume.Lock != nil {
		log = log.With(
			slog.Any("oldLockUuid", v.Volume.Lock.Uuid),
			slog.Any("oldLockTime", v.Volume.Lock.LockTime.AsTime().String()),
		)
	}

	allow := false
	lockUuid := ""
	if v.Volume.Lock == nil {
		allow = true
		lockUuid = uuid.NewString()
		log = log.With(slog.Any("newLockUuid", lockUuid))
		log.Info("locking volume")
	} else {
		if v.Volume.Lock.Uuid == req.GetPrevLockUuid() {
			allow = true
			lockUuid = v.Volume.Lock.Uuid
			log.Info("refreshing lock")
		} else if v.Volume.Lock.LockTime.AsTime().Add(lockTimeout).Before(time.Now()) {
			allow = true
			lockUuid = uuid.NewString()
			log = log.With(slog.Any("newLockUuid", lockUuid))
			log.Info("old lock expired, re-locking")
		}
	}
	if !allow {
		return nil, fmt.Errorf("volume is already locked")
	}

	v.Volume.Lock = &dproto.VolumeLock{
		Uuid:     lockUuid,
		LockTime: timestamppb.Now(),
	}

	err = v.Update()
	if err != nil {
		return nil, err
	}

	return &dproto.VolumeLockReply{
		Volume: v.Volume,
	}, nil
}
