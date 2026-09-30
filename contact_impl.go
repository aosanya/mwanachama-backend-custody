package mwanachamacustody

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

type ContactStore struct {
	st    *store
	clock Clock
}

func NewContactStore(db *gorm.DB, s *spec.Spec, clock Clock) (*ContactStore, error) {
	st, err := newStore(db, s)
	if err != nil {
		return nil, fmt.Errorf("NewContactStore: %w", err)
	}
	if clock == nil {
		clock = SystemClock
	}
	return &ContactStore{st: st, clock: clock}, nil
}

func (s *ContactStore) Create(ctx context.Context, r models.Read) (models.Read, error) {
	if r.ReadAt == "" {
		r.ReadAt = models.FormatTime(s.clock())
	}
	if err := check(s.st.Object(RoleContactRead), r); err != nil {
		return models.Read{}, err
	}
	if r.ID == "" {
		r.ID = specstore.NewID()
	}
	if err := s.st.Insert(ctx, RoleContactRead, r); err != nil {
		return models.Read{}, classify(err)
	}
	return r, nil
}

func (s *ContactStore) Get(ctx context.Context, id string) (models.Read, error) {
	var out models.Read
	if err := s.st.Take(s.st.Query(ctx, RoleContactRead).Where("id = ?", id), RoleContactRead, &out, models.ErrNotFound); err != nil {
		return models.Read{}, classify(err)
	}
	return out, nil
}

func (s *ContactStore) ListForSubject(ctx context.Context, actorID string) ([]models.Read, error) {
	return s.queryReads(s.st.Query(ctx, RoleContactRead).
		Where("actor_id = ?", actorID).
		Order("read_at DESC, id DESC"))
}

func (s *ContactStore) ListByOperator(ctx context.Context, operatorID string) ([]models.Read, error) {
	return s.queryReads(s.st.Query(ctx, RoleContactRead).
		Where("read_by = ?", operatorID).
		Order("read_at DESC, id DESC"))
}

func (s *ContactStore) queryReads(q *gorm.DB) ([]models.Read, error) {
	out, err := specstore.List[models.Read](s.st, q, RoleContactRead)
	if err != nil {
		return nil, classify(err)
	}
	return out, nil
}

var _ models.ContactRepository = (*ContactStore)(nil)
