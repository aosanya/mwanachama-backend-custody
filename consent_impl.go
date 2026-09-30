package mwanachamacustody

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

type ConsentStore struct {
	db *gorm.DB
	st *store
}

func NewConsentStore(db *gorm.DB, s *spec.Spec) (*ConsentStore, error) {
	st, err := newStore(db, s)
	if err != nil {
		return nil, fmt.Errorf("NewConsentStore: %w", err)
	}
	return &ConsentStore{db: db, st: st}, nil
}

func (s *ConsentStore) CreateVersion(ctx context.Context, v models.TextVersion) (models.TextVersion, error) {
	v.PublishedAt, v.PublishedBy, v.SupersededAt = nil, "", nil
	v.CopiedMechanicsID = ""
	if err := check(s.st.Object(RoleConsentVersion), v); err != nil {
		return models.TextVersion{}, err
	}
	if v.ID == "" {
		v.ID = specstore.NewID()
	}
	if err := s.st.Insert(ctx, RoleConsentVersion, v); err != nil {
		return models.TextVersion{}, classify(err)
	}
	return v, nil
}

func (s *ConsentStore) PublishVersion(ctx context.Context, id, publishedBy string, now string) (models.TextVersion, error) {
	if now == "" {
		now = models.Now()
	}
	var out models.TextVersion
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		table := s.st.Table(RoleConsentVersion)
		var current models.TextVersion
		if err := s.st.Take(tx.Table(table).Where("id = ?", id), RoleConsentVersion, &current, models.ErrNotFound); err != nil {
			return err
		}
		if current.PublishedAt != nil {
			return models.ErrAlreadyPublished
		}
		if err := tx.Table(table).
			Where("scope = ? AND language = ? AND published_at IS NOT NULL AND superseded_at IS NULL AND id != ?",
				string(current.Scope), string(current.Language), id).
			Update("superseded_at", now).Error; err != nil {
			return err
		}
		if err := tx.Table(table).Where("id = ?", id).
			Updates(map[string]any{"published_at": now, "published_by": publishedBy}).Error; err != nil {
			return err
		}
		return s.st.Take(tx.Table(table).Where("id = ?", id), RoleConsentVersion, &out, models.ErrNotFound)
	})
	if err != nil {
		if errors.Is(err, models.ErrNotFound) || errors.Is(err, models.ErrAlreadyPublished) {
			return models.TextVersion{}, err
		}
		return models.TextVersion{}, classify(err)
	}
	return out, nil
}

func (s *ConsentStore) GetVersion(ctx context.Context, id string) (models.TextVersion, error) {
	var out models.TextVersion
	if err := s.st.Take(s.st.Query(ctx, RoleConsentVersion).Where("id = ?", id), RoleConsentVersion, &out, models.ErrNotFound); err != nil {
		return models.TextVersion{}, classify(err)
	}
	return out, nil
}

func (s *ConsentStore) GetInForce(ctx context.Context, scope models.ConsentScope, language models.Language) (models.TextVersion, error) {
	var out models.TextVersion
	q := s.st.Query(ctx, RoleConsentVersion).
		Where("scope = ? AND language = ? AND published_at IS NOT NULL AND superseded_at IS NULL", string(scope), string(language)).
		Order("published_at DESC")
	if err := s.st.Take(q, RoleConsentVersion, &out, models.ErrNotFound); err != nil {
		return models.TextVersion{}, classify(err)
	}
	return out, nil
}

func (s *ConsentStore) ListVersions(ctx context.Context, scope models.ConsentScope, language models.Language) ([]models.TextVersion, error) {
	q := s.st.Query(ctx, RoleConsentVersion).
		Where("scope = ? AND language = ?", string(scope), string(language))
	if s.db.Dialector.Name() == "postgres" {
		out, err := specstore.List[models.TextVersion](s.st, q.Order("published_at DESC NULLS LAST, id"), RoleConsentVersion)
		if err != nil {
			return nil, classify(err)
		}
		return out, nil
	}
	out, err := specstore.List[models.TextVersion](s.st, q, RoleConsentVersion)
	if err != nil {
		return nil, classify(err)
	}
	sortTextVersionsNewestFirst(out)
	return out, nil
}

func sortTextVersionsNewestFirst(out []models.TextVersion) {
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && textVersionLess(out[j], out[j-1]); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
}

func textVersionLess(a, b models.TextVersion) bool {
	switch {
	case a.PublishedAt == nil && b.PublishedAt == nil:
		return a.ID < b.ID
	case a.PublishedAt == nil:
		return false
	case b.PublishedAt == nil:
		return true
	case *a.PublishedAt == *b.PublishedAt:
		return a.ID < b.ID
	default:
		return *a.PublishedAt > *b.PublishedAt
	}
}

func (s *ConsentStore) CreateRecord(ctx context.Context, r models.Record) (models.Record, error) {
	if err := s.mustExist(ctx, r.MechanicsVersionID, "mechanics_version_id"); err != nil {
		return models.Record{}, err
	}
	if r.AppendixVersionID != "" {
		if err := s.mustExist(ctx, r.AppendixVersionID, "appendix_version_id"); err != nil {
			return models.Record{}, err
		}
	}
	if err := check(s.st.Object(RoleConsentRecord), r); err != nil {
		return models.Record{}, err
	}
	if r.ID == "" {
		r.ID = specstore.NewID()
	}
	if err := s.st.Insert(ctx, RoleConsentRecord, r); err != nil {
		return models.Record{}, classify(err)
	}
	return r, nil
}

func (s *ConsentStore) mustExist(ctx context.Context, id, field string) error {
	var n int64
	if err := s.st.Query(ctx, RoleConsentVersion).Where("id = ?", id).Count(&n).Error; err != nil {
		return classify(err)
	}
	if n == 0 {
		return reference(field)
	}
	return nil
}

func (s *ConsentStore) GetRecord(ctx context.Context, id string) (models.Record, error) {
	var out models.Record
	if err := s.st.Take(s.st.Query(ctx, RoleConsentRecord).Where("id = ?", id), RoleConsentRecord, &out, models.ErrNotFound); err != nil {
		return models.Record{}, classify(err)
	}
	return out, nil
}

func (s *ConsentStore) ListForActor(ctx context.Context, actorID string) ([]models.Record, error) {
	out, err := specstore.List[models.Record](s.st,
		s.st.Query(ctx, RoleConsentRecord).Where("actor_id = ?", actorID).Order("agreed_at DESC, id DESC"),
		RoleConsentRecord)
	if err != nil {
		return nil, classify(err)
	}
	return out, nil
}

func (s *ConsentStore) CurrentForActor(ctx context.Context, actorID string) (models.Record, error) {
	var out models.Record
	q := s.st.Query(ctx, RoleConsentRecord).Where("actor_id = ?", actorID).Order("agreed_at DESC, id DESC")
	if err := s.st.Take(q, RoleConsentRecord, &out, models.ErrNotFound); err != nil {
		return models.Record{}, classify(err)
	}
	return out, nil
}

var _ models.ConsentRepository = (*ConsentStore)(nil)
