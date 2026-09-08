// contact_impl.go — GORM-backed models.ContactRepository implementation.
package mwanachamacustody

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-custody/gormstore"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

// ContactStore is the GORM implementation of [models.ContactRepository].
type ContactStore struct {
	db     *gorm.DB
	tables TableNames
	clock  Clock
}

// NewContactStore constructs a ContactStore. clock defaults to
// [SystemClock] when nil.
func NewContactStore(db *gorm.DB, t TableNames, clock Clock) (*ContactStore, error) {
	if db == nil {
		return nil, fmt.Errorf("NewContactStore: db must not be nil")
	}
	if clock == nil {
		clock = SystemClock
	}
	return &ContactStore{db: db, tables: t, clock: clock}, nil
}

func (s *ContactStore) Create(ctx context.Context, r models.Read) (models.Read, error) {
	if r.ReadAt.IsZero() {
		r.ReadAt = s.clock()
	}
	row := gormstore.ReadToRow(r)
	if err := s.db.WithContext(ctx).Table(s.tables.ContactRead).Create(&row).Error; err != nil {
		return models.Read{}, classify(err)
	}
	return gormstore.ReadFromRow(row), nil
}

// Get returns one read by id.
func (s *ContactStore) Get(ctx context.Context, id string) (models.Read, error) {
	var row gormstore.ReadRow
	err := s.db.WithContext(ctx).Table(s.tables.ContactRead).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Read{}, models.ErrNotFound
	}
	if err != nil {
		return models.Read{}, classify(err)
	}
	return gormstore.ReadFromRow(row), nil
}

func (s *ContactStore) ListForSubject(ctx context.Context, actorID string) ([]models.Read, error) {
	return s.query(ctx, "member_id = ?", actorID)
}

// ListByOperator returns every read operatorID performed, newest first.
func (s *ContactStore) ListByOperator(ctx context.Context, operatorID string) ([]models.Read, error) {
	return s.query(ctx, "read_by = ?", operatorID)
}

func (s *ContactStore) query(ctx context.Context, where string, arg string) ([]models.Read, error) {
	var rows []gormstore.ReadRow
	err := s.db.WithContext(ctx).Table(s.tables.ContactRead).
		Where(where, arg).Order("read_at DESC, id DESC").Find(&rows).Error
	if err != nil {
		return nil, classify(err)
	}
	out := make([]models.Read, 0, len(rows))
	for _, r := range rows {
		out = append(out, gormstore.ReadFromRow(r))
	}
	return out, nil
}

var _ models.ContactRepository = (*ContactStore)(nil)
