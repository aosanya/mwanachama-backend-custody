package mwanachamacustody

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

type ExportStore struct {
	st    *store
	clock Clock
}

func NewExportStore(db *gorm.DB, s *spec.Spec, clock Clock) (*ExportStore, error) {
	st, err := newStore(db, s)
	if err != nil {
		return nil, fmt.Errorf("NewExportStore: %w", err)
	}
	if clock == nil {
		clock = SystemClock
	}
	return &ExportStore{st: st, clock: clock}, nil
}

func pageOf(limit int) int {
	if limit <= 0 {
		return models.DefaultPage
	}
	return limit
}

func (s *ExportStore) Create(ctx context.Context, j models.Job) (models.Job, error) {
	if j.RequestedAt == "" {
		j.RequestedAt = models.FormatTime(s.clock())
	}
	if j.FileRemovedDueAt == "" && j.SnapshotAt != "" {
		due, err := models.DueAfter(j.SnapshotAt)
		if err != nil {
			return models.Job{}, err
		}
		j.FileRemovedDueAt = due
	}
	if err := j.Validate(); err != nil {
		return models.Job{}, err
	}
	if err := check(s.st.Object(RoleExportJob), j); err != nil {
		return models.Job{}, err
	}
	if j.ID == "" {
		j.ID = specstore.NewID()
	}
	if err := s.st.Insert(ctx, RoleExportJob, j); err != nil {
		return models.Job{}, classify(err)
	}
	return j, nil
}

func (s *ExportStore) Get(ctx context.Context, id string) (models.Job, error) {
	var out models.Job
	if err := s.st.Take(s.st.Query(ctx, RoleExportJob).Where("id = ?", id), RoleExportJob, &out, models.ErrNotFound); err != nil {
		return models.Job{}, classify(err)
	}
	return out, nil
}

func (s *ExportStore) ListOrganization(ctx context.Context, limit int) ([]models.Job, error) {
	return s.queryJobs(s.st.Query(ctx, RoleExportJob).
		Where("scope = ?", string(models.ScopeOrganization)).
		Order("requested_at DESC, id DESC").Limit(pageOf(limit)))
}

func (s *ExportStore) ListForActor(ctx context.Context, actorID string, limit int) ([]models.Job, error) {
	return s.queryJobs(s.st.Query(ctx, RoleExportJob).
		Where("scope = ? AND subject_actor_id = ?", string(models.ScopeActor), actorID).
		Order("requested_at DESC, id DESC").Limit(pageOf(limit)))
}

func (s *ExportStore) queryJobs(q *gorm.DB) ([]models.Job, error) {
	out, err := specstore.List[models.Job](s.st, q, RoleExportJob)
	if err != nil {
		return nil, classify(err)
	}
	return out, nil
}

// Progress records a build's advance. ResumedFromPct is write-once and the
// guard is in the UPDATE's WHERE clause rather than a read-then-write, so
// two workers racing cannot both believe they recorded the first figure.
func (s *ExportStore) Progress(ctx context.Context, id string, p models.ProgressUpdate) (models.Job, error) {
	if err := p.Validate(); err != nil {
		return models.Job{}, err
	}
	updates := map[string]any{}
	if p.Status != "" {
		updates["status"] = string(p.Status)
	}
	if p.Pct != nil {
		updates["progress_pct"] = *p.Pct
	}
	if p.ResumedFrom != nil {
		updates["resumed_from_pct"] = *p.ResumedFrom
	}
	if p.RowCounts != nil {
		counts, err := marshalDocument(p.RowCounts)
		if err != nil {
			return models.Job{}, err
		}
		updates["row_counts"] = counts
	}
	if len(updates) == 0 {
		return s.Get(ctx, id)
	}
	q := s.st.Query(ctx, RoleExportJob).Where("id = ?", id)
	if p.ResumedFrom != nil {
		q = q.Where("resumed_from_pct IS NULL")
	}
	tx := q.Updates(updates)
	if tx.Error != nil {
		return models.Job{}, classify(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return models.Job{}, s.explainNoRows(ctx, id, p.ResumedFrom != nil)
	}
	return s.Get(ctx, id)
}

func (s *ExportStore) explainNoRows(ctx context.Context, id string, wroteResume bool) error {
	if !wroteResume {
		return models.ErrNotFound
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return models.ErrResumeAlreadyRecorded
}

func (s *ExportStore) Complete(ctx context.Context, id string, c models.Completion) (models.Job, error) {
	if err := c.Validate(); err != nil {
		return models.Job{}, err
	}
	updates := map[string]any{
		"status":       string(models.StatusCompleted),
		"progress_pct": 100,
		"bytes":        c.Bytes,
		"checksum":     c.Checksum,
		"storage_path": c.StoragePath,
	}
	if c.RowCounts != nil {
		counts, err := marshalDocument(c.RowCounts)
		if err != nil {
			return models.Job{}, err
		}
		updates["row_counts"] = counts
	}
	tx := s.st.Query(ctx, RoleExportJob).Where("id = ?", id).Updates(updates)
	if tx.Error != nil {
		return models.Job{}, classify(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return models.Job{}, models.ErrNotFound
	}
	return s.Get(ctx, id)
}

// MarkFileRemoved stamps file_removed_at and keeps the row. There is no
// DELETE anywhere in this file, which is how "the row is never deleted"
// survives: no caller can express it.
func (s *ExportStore) MarkFileRemoved(ctx context.Context, id string, at string) (models.Job, error) {
	if at == "" {
		at = models.FormatTime(s.clock())
	}
	tx := s.st.Query(ctx, RoleExportJob).Where("id = ?", id).
		Update("file_removed_at", at)
	if tx.Error != nil {
		return models.Job{}, classify(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return models.Job{}, models.ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *ExportStore) DueForRemoval(ctx context.Context, asOf string, limit int) ([]models.Job, error) {
	if asOf == "" {
		asOf = models.FormatTime(s.clock())
	}
	return s.queryJobs(s.st.Query(ctx, RoleExportJob).
		Where("file_removed_at IS NULL AND file_removed_due_at <= ?", asOf).
		Order("file_removed_due_at ASC, id ASC").Limit(pageOf(limit)))
}

var _ models.ExportRepository = (*ExportStore)(nil)
