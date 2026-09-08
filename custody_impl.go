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

// CustodyStore is the GORM implementation of [models.CustodyRepository].
//
// **There is no UPDATE and no DELETE statement anywhere in this file.** That
// is the append-only invariant, and it is a property of the file rather than
// a rule enforced inside one — the source object files say `update: nobody`
// and `delete: nobody`, and a store with no such statement cannot be talked
// into one.
type CustodyStore struct {
	db     *gorm.DB
	tables TableNames
	clock  Clock
}

// NewCustodyStore constructs a CustodyStore backed by db, reading and
// writing the tables named by t. Callers must run [Migrate] against the
// same db and t before use. clock defaults to [SystemClock] when nil.
func NewCustodyStore(db *gorm.DB, t TableNames, clock Clock) (*CustodyStore, error) {
	if db == nil {
		return nil, fmt.Errorf("NewCustodyStore: db must not be nil")
	}
	if clock == nil {
		clock = SystemClock
	}
	return &CustodyStore{db: db, tables: t, clock: clock}, nil
}

func (s *CustodyStore) AppendAct(ctx context.Context, e models.StructureActLogEntry) (models.StructureActLogEntry, error) {
	class, err := models.ActClassOf(e.Kind)
	if err != nil {
		return models.StructureActLogEntry{}, err
	}
	if e.StructureID == "" {
		return models.StructureActLogEntry{}, models.ErrStructureRequired
	}
	e.Class = class
	if e.OccurredAt.IsZero() {
		e.OccurredAt = s.clock()
	}
	row, err := gormstore.StructureActLogEntryToRow(e)
	if err != nil {
		return models.StructureActLogEntry{}, err
	}
	if err := s.db.WithContext(ctx).Table(s.tables.StructureActLog).Create(&row).Error; err != nil {
		return models.StructureActLogEntry{}, classify(err)
	}
	return gormstore.StructureActLogEntryFromRow(row)
}

func (s *CustodyStore) ListActs(ctx context.Context, structures []string, class models.ActClass, limit int) ([]models.StructureActLogEntry, error) {
	if limit <= 0 {
		limit = models.DefaultPage
	}
	q := s.db.WithContext(ctx).Table(s.tables.StructureActLog)
	if len(structures) > 0 {
		q = q.Where("chapter_id IN ?", structures)
	}
	if class != "" {
		q = q.Where("class = ?", string(class))
	}
	var rows []gormstore.StructureActLogEntryRow
	if err := q.Order("occurred_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, classify(err)
	}
	out := make([]models.StructureActLogEntry, 0, len(rows))
	for _, r := range rows {
		e, err := gormstore.StructureActLogEntryFromRow(r)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// CountActsByClass returns the per-class tally over one query, so the counts
// sum to the same total a concurrent ListActs would see.
func (s *CustodyStore) CountActsByClass(ctx context.Context, structures []string, since time.Time) (map[models.ActClass]int, error) {
	q := s.db.WithContext(ctx).Table(s.tables.StructureActLog)
	if len(structures) > 0 {
		q = q.Where("chapter_id IN ?", structures)
	}
	if !since.IsZero() {
		q = q.Where("occurred_at >= ?", since)
	}
	var rows []struct {
		Class string
		N     int
	}
	if err := q.Select("class, count(*) as n").Group("class").Scan(&rows).Error; err != nil {
		return nil, classify(err)
	}
	out := map[models.ActClass]int{}
	for _, r := range rows {
		out[models.ActClass(r.Class)] = r.N
	}
	return out, nil
}

// AppendEvent writes one custody row, deriving Chip from Kind before
// anything is written, for AppendAct's reason.
func (s *CustodyStore) AppendEvent(ctx context.Context, e models.Entry) (models.Entry, error) {
	chip, err := models.EventChipOf(e.Kind)
	if err != nil {
		return models.Entry{}, err
	}
	e.Chip = chip
	if e.OccurredAt.IsZero() {
		e.OccurredAt = s.clock()
	}
	row, err := gormstore.EntryToRow(e)
	if err != nil {
		return models.Entry{}, err
	}
	if err := s.db.WithContext(ctx).Table(s.tables.CustodyEvent).Create(&row).Error; err != nil {
		return models.Entry{}, classify(err)
	}
	return gormstore.EntryFromRow(row)
}

// ListEvents returns custody rows newest first, optionally narrowed to a chip.
func (s *CustodyStore) ListEvents(ctx context.Context, chip models.EventChip, limit int) ([]models.Entry, error) {
	if limit <= 0 {
		limit = models.DefaultPage
	}
	q := s.db.WithContext(ctx).Table(s.tables.CustodyEvent)
	if chip != "" {
		q = q.Where("chip = ?", string(chip))
	}
	var rows []gormstore.EntryRow
	if err := q.Order("occurred_at DESC, id DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, classify(err)
	}
	out := make([]models.Entry, 0, len(rows))
	for _, r := range rows {
		e, err := gormstore.EntryFromRow(r)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// CountEventsByChip returns the per-chip tally over one query, for
// CountActsByClass's reason.
func (s *CustodyStore) CountEventsByChip(ctx context.Context, since time.Time) (map[models.EventChip]int, error) {
	q := s.db.WithContext(ctx).Table(s.tables.CustodyEvent)
	if !since.IsZero() {
		q = q.Where("occurred_at >= ?", since)
	}
	var rows []struct {
		Chip string
		N    int
	}
	if err := q.Select("chip, count(*) as n").Group("chip").Scan(&rows).Error; err != nil {
		return nil, classify(err)
	}
	out := map[models.EventChip]int{}
	for _, r := range rows {
		out[models.EventChip(r.Chip)] = r.N
	}
	return out, nil
}

// GetEvent returns one custody row by id.
func (s *CustodyStore) GetEvent(ctx context.Context, id int64) (models.Entry, error) {
	var row gormstore.EntryRow
	err := s.db.WithContext(ctx).Table(s.tables.CustodyEvent).Where("id = ?", id).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Entry{}, models.ErrNotFound
	}
	if err != nil {
		return models.Entry{}, classify(err)
	}
	return gormstore.EntryFromRow(row)
}

var _ models.CustodyRepository = (*CustodyStore)(nil)
