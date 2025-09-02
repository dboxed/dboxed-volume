package volumes

import (
	"context"
	"log/slog"
	"slices"
	"time"

	"github.com/danielgtaylor/huma/v2"
	"github.com/dboxed/dboxed-common/db/querier"
	"github.com/dboxed/dboxed-common/huma_utils"
	"github.com/dboxed/dboxed-common/util"
	"github.com/dboxed/dboxed-volume/pkg/config"
	"github.com/dboxed/dboxed-volume/pkg/db/dmodel"
	"github.com/dboxed/dboxed-volume/pkg/server/models"
	"github.com/dboxed/dboxed-volume/pkg/server/resources/repositories"
	"github.com/dboxed/dboxed-volume/pkg/volume"
	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
)

type Volumes struct {
}

func New(config config.Config) *Volumes {
	s := &Volumes{}
	return s
}

func (s *Volumes) Init(api huma.API) error {
	repoGroup := huma.NewGroup(api, "/v1/repositories/{repositoryId}")
	repoGroup.UseMiddleware(repositories.RepositoryMiddleware(api))

	huma.Post(repoGroup, "/volumes", s.restCreateVolume)
	huma.Get(repoGroup, "/volumes", s.restListVolumes)
	huma.Get(repoGroup, "/volumes/{id}", s.restGetVolume)
	huma.Get(repoGroup, "/volumes/by-name/{volumeName}", s.restGetVolumeByName)
	huma.Delete(repoGroup, "/volumes/{id}", s.restDeleteVolume)

	huma.Post(repoGroup, "/volumes/{id}/lock", s.restLockVolume)

	return nil
}

func (s *Volumes) restCreateVolume(ctx context.Context, i *huma_utils.JsonBody[models.CreateVolume]) (*huma_utils.JsonBody[models.Volume], error) {
	q := querier.GetQuerier(ctx)
	r := repositories.GetRepository(ctx)

	if i.Body.FsSize <= humanize.MiByte {
		return nil, huma.Error400BadRequest("fsSize is too small")
	}
	if !slices.Contains(volume.AllowedFsTypes, i.Body.FsType) {
		return nil, huma.Error400BadRequest("unsupported or invalid fsType")
	}
	err := util.CheckName(i.Body.Name)
	if err != nil {
		return nil, err
	}

	v := dmodel.Volume{
		Uuid:         uuid.NewString(),
		Name:         i.Body.Name,
		RepositoryID: r.ID,
		FsSize:       i.Body.FsSize,
		FsType:       i.Body.FsType,
	}

	err = v.Create(q)
	if err != nil {
		return nil, err
	}

	return huma_utils.NewJsonBody(models.VolumeFromDB(v)), nil
}

func (s *Volumes) restListVolumes(ctx context.Context, i *struct{}) (*huma_utils.List[models.Volume], error) {
	q := querier.GetQuerier(ctx)
	r := repositories.GetRepository(ctx)

	l, err := dmodel.ListVolumesForRepository(q, r.ID, true)
	if err != nil {
		return nil, err
	}

	var ret []models.Volume
	for _, r := range l {
		mm := models.VolumeFromDB(r)
		ret = append(ret, mm)
	}
	return huma_utils.NewList(ret, len(ret)), nil
}

func (s *Volumes) restGetVolume(c context.Context, i *huma_utils.IdByPath) (*huma_utils.JsonBody[models.Volume], error) {
	q := querier.GetQuerier(c)
	r := repositories.GetRepository(c)

	v, err := dmodel.GetVolumeById(q, &r.ID, i.Id, true)
	if err != nil {
		return nil, err
	}

	m := models.VolumeFromDB(*v)
	return huma_utils.NewJsonBody(m), nil
}

type VolumeName struct {
	VolumeName string `path:"volumeName"`
}

func (s *Volumes) restGetVolumeByName(c context.Context, i *VolumeName) (*huma_utils.JsonBody[models.Volume], error) {
	q := querier.GetQuerier(c)
	r := repositories.GetRepository(c)

	v, err := dmodel.GetVolumeByName(q, r.ID, i.VolumeName, true)
	if err != nil {
		return nil, err
	}

	m := models.VolumeFromDB(*v)
	return huma_utils.NewJsonBody(m), nil
}

func (s *Volumes) restDeleteVolume(c context.Context, i *huma_utils.IdByPath) (*huma_utils.Empty, error) {
	q := querier.GetQuerier(c)

	err := dmodel.SoftDeleteWithConstraintsByIds[dmodel.Volume](q, i.Id)
	if err != nil {
		return nil, err
	}

	return &huma_utils.Empty{}, nil
}

type restLockVolume struct {
	huma_utils.IdByPath
	Body models.VolumeLockRequest
}

func (s *Volumes) restLockVolume(c context.Context, i *restLockVolume) (*huma_utils.JsonBody[models.Volume], error) {
	q := querier.GetQuerier(c)
	r := repositories.GetRepository(c)

	v, err := dmodel.GetVolumeById(q, &r.ID, i.Id, true)
	if err != nil {
		return nil, err
	}

	log := slog.With(slog.Any("repoId", r.ID), slog.Any("volId", v.ID))

	strConv := func(s *string) string {
		if s == nil {
			return ""
		}
		return *s
	}

	lockTimeout := time.Minute * 5
	allow := false
	lockUuid := ""
	if v.LockId == nil {
		allow = true
		lockUuid = uuid.NewString()
		log = log.With(slog.Any("newLockId", lockUuid))
		log.Info("locking volume")
	} else {
		if strConv(v.LockId) == strConv(i.Body.PrevLockId) {
			allow = true
			lockUuid = *v.LockId
			log.Info("refreshing lock")
		} else if *v.LockTime+int64(lockTimeout.Seconds()) < time.Now().Unix() {
			allow = true
			lockUuid = uuid.NewString()
			log = log.With(slog.Any("newLockId", lockUuid))
			log.Info("old lock expired, re-locking")
		}
	}
	if !allow {
		return nil, huma.Error409Conflict("volume is already locked")
	}

	err = v.UpdateLock(q, lockUuid, time.Now())
	if err != nil {
		return nil, err
	}

	m := models.VolumeFromDB(*v)
	return huma_utils.NewJsonBody(m), nil
}
