// export_impl.go — GORM-backed models.ExportRepository implementation.
package mwanachamacustody

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-custody/gormstore"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

// ExportStore is the GORM implementation of [models.ExportRepository].
//
// **Every write calls the domain's Validate before it reaches the database.**
// That is not redundancy for its own sake: it is what makes the memory-backed
// (sqlite, in this repo's tests) and Postgres-backed callers get the same
// refusal, in the same words.
type ExportStore struct {
	db     *gorm.DB
	tables TableNames
	clock  Clock
}

// NewExportStore constructs an ExportStore. clock defaults to [SystemClock]
// when nil and fills RequestedAt on Create when the caller left it unset —
// it never fills SnapshotAt, which the caller must supply itself (G43).
func NewExportStore(db *gorm.DB, t TableNames, clock Clock) (*ExportStore, error) {
	if db == nil {
		return nil, fmt.Errorf("NewExportStore: db must not be nil")
	}
	if clock == nil {
		clock = SystemClock
	}
	return &ExportStore{db: db, tables: t, clock: clock}, nil
}

func pageOf(limit int) int {
	if limit <= 0 {
		return models.DefaultPage
	}
	return limit
}

// Create inserts one export_job row.
func (s *ExportStore) Create(ctx context.Context, j models.Job) (models.Job, error) {
	if j.RequestedAt.IsZero() {
		j.RequestedAt = s.clock()
	}
	if j.FileRemovedDueAt.IsZero() && !j.SnapshotAt.IsZero() {
		j.FileRemovedDueAt = models.DueAt(j.SnapshotAt)
	}
	if err := j.Validate(); err != nil {
		return models.Job{}, err
	}
	row, err := gormstore.JobToRow(j)
	if err != nil {
		return models.Job{}, err
	}
	if err := s.db.WithContext(ctx).Table(s.tables.ExportJob).Create(&row).Error; err != nil {
		return models.Job{}, classify(err)
	}
	return gormstore.JobFromRow(row)
}

// Get returns one job by id.
func (s *ExportStore) Get(ctx context.Context, id string) (models.Job, error) {
	var row gormstore.JobRow
	err := s.db.WithContext(ctx).Table(s.tables.ExportJob).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Job{}, models.ErrNotFound
	}
	if err != nil {
		return models.Job{}, classify(err)
	}
	return gormstore.JobFromRow(row)
}

func (s *ExportStore) ListOrganization(ctx context.Context, limit int) ([]models.Job, error) {
	return s.queryJobs(ctx, s.db.WithContext(ctx).Table(s.tables.ExportJob).
		Where("scope = ?", string(models.ScopeOrganization)).
		Order("requested_at DESC, id DESC").Limit(pageOf(limit)))
}

func (s *ExportStore) ListForActor(ctx context.Context, actorID string, limit int) ([]models.Job, error) {
	return s.queryJobs(ctx, s.db.WithContext(ctx).Table(s.tables.ExportJob).
		Where("scope = ? AND subject_member_id = ?", string(models.ScopeActor), actorID).
		Order("requested_at DESC, id DESC").Limit(pageOf(limit)))
}

func (s *ExportStore) queryJobs(_ context.Context, q *gorm.DB) ([]models.Job, error) {
	var rows []gormstore.JobRow
	if err := q.Find(&rows).Error; err != nil {
		return nil, classify(err)
	}
	out := make([]models.Job, 0, len(rows))
	for _, r := range rows {
		j, err := gormstore.JobFromRow(r)
		if err != nil {
			return nil, err
		}
		out = append(out, j)
	}
	return out, nil
}

// Progress records a build's advance. **ResumedFromPct is write-once, and the
// guard is in the UPDATE's WHERE clause rather than in a read-then-write** —
// a second resume matches no row, so two workers racing cannot both believe
// they recorded the first figure.
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
		counts, err := gormstore.MarshalCounts(p.RowCounts)
		if err != nil {
			return models.Job{}, err
		}
		updates["row_counts"] = counts
	}
	if len(updates) == 0 {
		return s.Get(ctx, id)
	}
	q := s.db.WithContext(ctx).Table(s.tables.ExportJob).Where("id = ?", id)
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

// explainNoRows tells the two reasons an UPDATE matched nothing apart: an
// unknown id, or a write-once violation on an id that does exist.
func (s *ExportStore) explainNoRows(ctx context.Context, id string, wroteResume bool) error {
	if !wroteResume {
		return models.ErrNotFound
	}
	if _, err := s.Get(ctx, id); err != nil {
		return err
	}
	return models.ErrResumeAlreadyRecorded
}

// Complete records the finish: bytes, checksum, storage path.
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
		counts, err := gormstore.MarshalCounts(c.RowCounts)
		if err != nil {
			return models.Job{}, err
		}
		updates["row_counts"] = counts
	}
	tx := s.db.WithContext(ctx).Table(s.tables.ExportJob).Where("id = ?", id).Updates(updates)
	if tx.Error != nil {
		return models.Job{}, classify(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return models.Job{}, models.ErrNotFound
	}
	return s.Get(ctx, id)
}

// MarkFileRemoved stamps file_removed_at and keeps the row — there is no
// DELETE anywhere in this file, which is how "the row is never deleted"
// survives the port: no caller can express it.
func (s *ExportStore) MarkFileRemoved(ctx context.Context, id string, at time.Time) (models.Job, error) {
	if at.IsZero() {
		at = s.clock()
	}
	tx := s.db.WithContext(ctx).Table(s.tables.ExportJob).Where("id = ?", id).
		Update("file_removed_at", at)
	if tx.Error != nil {
		return models.Job{}, classify(tx.Error)
	}
	if tx.RowsAffected == 0 {
		return models.Job{}, models.ErrNotFound
	}
	return s.Get(ctx, id)
}

// DueForRemoval returns jobs past their due date whose file is still there,
// oldest first — the sweep that makes a stalled worker visible as rows
// sitting past their own stated date.
func (s *ExportStore) DueForRemoval(ctx context.Context, asOf time.Time, limit int) ([]models.Job, error) {
	if asOf.IsZero() {
		asOf = s.clock()
	}
	return s.queryJobs(ctx, s.db.WithContext(ctx).Table(s.tables.ExportJob).
		Where("file_removed_at IS NULL AND file_removed_due_at <= ?", asOf).
		Order("file_removed_due_at ASC, id ASC").Limit(pageOf(limit)))
}

var _ models.ExportRepository = (*ExportStore)(nil)
