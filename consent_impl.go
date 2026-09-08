// consent_impl.go — GORM-backed models.ConsentRepository implementation.
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

// ConsentStore is the GORM implementation of [models.ConsentRepository].
type ConsentStore struct {
	db     *gorm.DB
	tables TableNames
}

// NewConsentStore constructs a ConsentStore.
func NewConsentStore(db *gorm.DB, t TableNames) (*ConsentStore, error) {
	if db == nil {
		return nil, fmt.Errorf("NewConsentStore: db must not be nil")
	}
	return &ConsentStore{db: db, tables: t}, nil
}

// CreateVersion inserts a new, unpublished version row. Scope/Version/
// Language is a unique index on the row (gormstore.TextVersionRow), so a
// second row with the same triple returns ErrConflict via classify rather
// than a pre-check racing a concurrent insert.
func (s *ConsentStore) CreateVersion(ctx context.Context, v models.TextVersion) (models.TextVersion, error) {
	// DEV-1174 · a version is created unpublished, whatever the caller
	// supplied: PublishVersion is the only writer of these three fields, and
	// the in-force rule (GetInForce) is derived from two of them with no
	// guard of its own — a caller-supplied value here would put text into
	// force without the act that authorises it.
	v.PublishedAt, v.PublishedBy, v.SupersededAt = nil, "", nil
	v.CopiedMechanicsID = ""
	row, err := gormstore.TextVersionToRow(v)
	if err != nil {
		return models.TextVersion{}, err
	}
	if err := s.db.WithContext(ctx).Table(s.tables.ConsentTextVersion).Create(&row).Error; err != nil {
		return models.TextVersion{}, classify(err)
	}
	return gormstore.TextVersionFromRow(row)
}

// PublishVersion stamps PublishedAt/PublishedBy and supersedes whichever
// version was previously in force for the same (scope, language), inside
// one transaction so the two writes are atomic.
func (s *ConsentStore) PublishVersion(ctx context.Context, id, publishedBy string, now time.Time) (models.TextVersion, error) {
	var out models.TextVersion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row gormstore.TextVersionRow
		if err := tx.Table(s.tables.ConsentTextVersion).Where("id = ?", id).First(&row).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return models.ErrNotFound
			}
			return err
		}
		if row.PublishedAt != nil {
			return models.ErrAlreadyPublished
		}
		if err := tx.Table(s.tables.ConsentTextVersion).
			Where("scope = ? AND language = ? AND published_at IS NOT NULL AND superseded_at IS NULL AND id != ?",
				row.Scope, row.Language, id).
			Update("superseded_at", now).Error; err != nil {
			return err
		}
		if err := tx.Table(s.tables.ConsentTextVersion).Where("id = ?", id).
			Updates(map[string]any{"published_at": now, "published_by": publishedBy}).Error; err != nil {
			return err
		}
		if err := tx.Table(s.tables.ConsentTextVersion).Where("id = ?", id).First(&row).Error; err != nil {
			return err
		}
		v, err := gormstore.TextVersionFromRow(row)
		if err != nil {
			return err
		}
		out = v
		return nil
	})
	if err != nil {
		if errors.Is(err, models.ErrNotFound) || errors.Is(err, models.ErrAlreadyPublished) {
			return models.TextVersion{}, err
		}
		return models.TextVersion{}, classify(err)
	}
	return out, nil
}

// GetVersion returns one version by id.
func (s *ConsentStore) GetVersion(ctx context.Context, id string) (models.TextVersion, error) {
	var row gormstore.TextVersionRow
	err := s.db.WithContext(ctx).Table(s.tables.ConsentTextVersion).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.TextVersion{}, models.ErrNotFound
	}
	if err != nil {
		return models.TextVersion{}, classify(err)
	}
	return gormstore.TextVersionFromRow(row)
}

// GetInForce returns the version with the greatest published_at and a null
// superseded_at, within the given (scope, language).
func (s *ConsentStore) GetInForce(ctx context.Context, scope models.ConsentScope, language models.Language) (models.TextVersion, error) {
	var row gormstore.TextVersionRow
	err := s.db.WithContext(ctx).Table(s.tables.ConsentTextVersion).
		Where("scope = ? AND language = ? AND published_at IS NOT NULL AND superseded_at IS NULL", string(scope), string(language)).
		Order("published_at DESC").Limit(1).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.TextVersion{}, models.ErrNotFound
	}
	if err != nil {
		return models.TextVersion{}, classify(err)
	}
	return gormstore.TextVersionFromRow(row)
}

// ListVersions returns every version for a (scope, language), newest
// published_at first (unpublished rows last).
func (s *ConsentStore) ListVersions(ctx context.Context, scope models.ConsentScope, language models.Language) ([]models.TextVersion, error) {
	var rows []gormstore.TextVersionRow
	err := s.db.WithContext(ctx).Table(s.tables.ConsentTextVersion).
		Where("scope = ? AND language = ?", string(scope), string(language)).
		Order("published_at DESC NULLS LAST, id").Find(&rows).Error
	if err != nil {
		// sqlite (this repo's own tests) has no NULLS LAST syntax; fall back
		// to an unordered-by-null-first query and sort in Go. Postgres never
		// reaches this branch.
		if s.db.Dialector.Name() != "postgres" {
			return s.listVersionsPortable(ctx, scope, language)
		}
		return nil, classify(err)
	}
	return fromTextVersionRows(rows)
}

func (s *ConsentStore) listVersionsPortable(ctx context.Context, scope models.ConsentScope, language models.Language) ([]models.TextVersion, error) {
	var rows []gormstore.TextVersionRow
	if err := s.db.WithContext(ctx).Table(s.tables.ConsentTextVersion).
		Where("scope = ? AND language = ?", string(scope), string(language)).
		Find(&rows).Error; err != nil {
		return nil, classify(err)
	}
	out, err := fromTextVersionRows(rows)
	if err != nil {
		return nil, err
	}
	sortTextVersionsNewestFirst(out)
	return out, nil
}

func fromTextVersionRows(rows []gormstore.TextVersionRow) ([]models.TextVersion, error) {
	out := make([]models.TextVersion, 0, len(rows))
	for _, r := range rows {
		v, err := gormstore.TextVersionFromRow(r)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, nil
}

func sortTextVersionsNewestFirst(out []models.TextVersion) {
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && textVersionLess(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
}

// textVersionLess reports whether a sorts before b: greatest PublishedAt
// first, unpublished (nil) rows last, ties broken by id.
func textVersionLess(a, b models.TextVersion) bool {
	switch {
	case a.PublishedAt == nil && b.PublishedAt == nil:
		return a.ID < b.ID
	case a.PublishedAt == nil:
		return false
	case b.PublishedAt == nil:
		return true
	case a.PublishedAt.Equal(*b.PublishedAt):
		return a.ID < b.ID
	default:
		return a.PublishedAt.After(*b.PublishedAt)
	}
}

// CreateRecord inserts an immutable consent record. Both version references
// must already exist.
func (s *ConsentStore) CreateRecord(ctx context.Context, r models.Record) (models.Record, error) {
	if err := s.mustExist(ctx, r.MechanicsVersionID, "mechanics_version_id"); err != nil {
		return models.Record{}, err
	}
	if r.AppendixVersionID != "" {
		if err := s.mustExist(ctx, r.AppendixVersionID, "appendix_version_id"); err != nil {
			return models.Record{}, err
		}
	}
	row := gormstore.RecordToRow(r)
	if err := s.db.WithContext(ctx).Table(s.tables.ConsentRecord).Create(&row).Error; err != nil {
		return models.Record{}, classify(err)
	}
	return gormstore.RecordFromRow(row), nil
}

func (s *ConsentStore) mustExist(ctx context.Context, id, field string) error {
	var n int64
	if err := s.db.WithContext(ctx).Table(s.tables.ConsentTextVersion).Where("id = ?", id).Count(&n).Error; err != nil {
		return classify(err)
	}
	if n == 0 {
		return reference(field)
	}
	return nil
}

// GetRecord returns one record by id.
func (s *ConsentStore) GetRecord(ctx context.Context, id string) (models.Record, error) {
	var row gormstore.RecordRow
	err := s.db.WithContext(ctx).Table(s.tables.ConsentRecord).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Record{}, models.ErrNotFound
	}
	if err != nil {
		return models.Record{}, classify(err)
	}
	return gormstore.RecordFromRow(row), nil
}

func (s *ConsentStore) ListForActor(ctx context.Context, actorID string) ([]models.Record, error) {
	var rows []gormstore.RecordRow
	err := s.db.WithContext(ctx).Table(s.tables.ConsentRecord).
		Where("member_id = ?", actorID).Order("agreed_at DESC, id DESC").Find(&rows).Error
	if err != nil {
		return nil, classify(err)
	}
	out := make([]models.Record, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.RecordFromRow(r))
	}
	return out, nil
}

func (s *ConsentStore) CurrentForActor(ctx context.Context, actorID string) (models.Record, error) {
	var row gormstore.RecordRow
	err := s.db.WithContext(ctx).Table(s.tables.ConsentRecord).
		Where("member_id = ?", actorID).Order("agreed_at DESC, id DESC").Limit(1).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Record{}, models.ErrNotFound
	}
	if err != nil {
		return models.Record{}, classify(err)
	}
	return gormstore.RecordFromRow(row), nil
}

var _ models.ConsentRepository = (*ConsentStore)(nil)
