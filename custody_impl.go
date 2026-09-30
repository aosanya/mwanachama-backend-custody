package mwanachamacustody

import (
	"context"
	"fmt"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-shared/spec"
	"github.com/aosanya/mwanachama-backend-shared/specstore"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

type CustodyStore struct {
	db    *gorm.DB
	st    *store
	clock Clock
}

func NewCustodyStore(db *gorm.DB, s *spec.Spec, clock Clock) (*CustodyStore, error) {
	st, err := newStore(db, s)
	if err != nil {
		return nil, fmt.Errorf("NewCustodyStore: %w", err)
	}
	if clock == nil {
		clock = SystemClock
	}
	return &CustodyStore{db: db, st: st, clock: clock}, nil
}

func (s *CustodyStore) Classes() []models.ActClass {
	values := declaredValues(s.st.Object(RoleStructureAct), "class")
	out := make([]models.ActClass, 0, len(values))
	for _, v := range values {
		out = append(out, models.ActClass(v))
	}
	return out
}

func (s *CustodyStore) Chips() []models.EventChip {
	values := declaredValues(s.st.Object(RoleEvent), "chip")
	out := make([]models.EventChip, 0, len(values))
	for _, v := range values {
		out = append(out, models.EventChip(v))
	}
	return out
}

func (s *CustodyStore) ValidClass(class models.ActClass) bool {
	return checkField(s.st.Object(RoleStructureAct), "class", string(class)) == nil
}

func (s *CustodyStore) ValidChip(chip models.EventChip) bool {
	return checkField(s.st.Object(RoleEvent), "chip", string(chip)) == nil
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
	if e.OccurredAt == "" {
		e.OccurredAt = models.FormatTime(s.clock())
	}
	if err := check(s.st.Object(RoleStructureAct), e); err != nil {
		return models.StructureActLogEntry{}, err
	}
	if e.ID == 0 {
		next, err := nextSequenceID(s.db.WithContext(ctx), s.st.Table(RoleStructureAct), sequenceFor(RoleStructureAct))
		if err != nil {
			return models.StructureActLogEntry{}, err
		}
		e.ID = next
	}
	if err := s.st.Insert(ctx, RoleStructureAct, e); err != nil {
		return models.StructureActLogEntry{}, classify(err)
	}
	return e, nil
}

func actScope(q *gorm.DB, structures []string) *gorm.DB {
	if len(structures) > 0 {
		return q.Where("structure_id IN ?", structures)
	}
	return q
}

func (s *CustodyStore) ListActs(ctx context.Context, structures []string, class models.ActClass, limit int) ([]models.StructureActLogEntry, error) {
	q := actScope(s.st.Query(ctx, RoleStructureAct), structures)
	if class != "" {
		q = q.Where("class = ?", string(class))
	}
	out, err := specstore.List[models.StructureActLogEntry](s.st,
		q.Order("occurred_at DESC, id DESC").Limit(pageOf(limit)), RoleStructureAct)
	if err != nil {
		return nil, classify(err)
	}
	return out, nil
}

func (s *CustodyStore) CountActsByClass(ctx context.Context, structures []string, since string) (map[models.ActClass]int, error) {
	q := actScope(s.st.Query(ctx, RoleStructureAct), structures)
	if since != "" {
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

func (s *CustodyStore) AppendEvent(ctx context.Context, e models.Entry) (models.Entry, error) {
	chip, err := models.EventChipOf(e.Kind)
	if err != nil {
		return models.Entry{}, err
	}
	e.Chip = chip
	if e.OccurredAt == "" {
		e.OccurredAt = models.FormatTime(s.clock())
	}
	if err := check(s.st.Object(RoleEvent), e); err != nil {
		return models.Entry{}, err
	}
	if e.ID == 0 {
		next, err := nextSequenceID(s.db.WithContext(ctx), s.st.Table(RoleEvent), sequenceFor(RoleEvent))
		if err != nil {
			return models.Entry{}, err
		}
		e.ID = next
	}
	if err := s.st.Insert(ctx, RoleEvent, e); err != nil {
		return models.Entry{}, classify(err)
	}
	return e, nil
}

func (s *CustodyStore) ListEvents(ctx context.Context, chip models.EventChip, limit int) ([]models.Entry, error) {
	q := s.st.Query(ctx, RoleEvent)
	if chip != "" {
		q = q.Where("chip = ?", string(chip))
	}
	out, err := specstore.List[models.Entry](s.st,
		q.Order("occurred_at DESC, id DESC").Limit(pageOf(limit)), RoleEvent)
	if err != nil {
		return nil, classify(err)
	}
	return out, nil
}

func (s *CustodyStore) CountEventsByChip(ctx context.Context, since string) (map[models.EventChip]int, error) {
	q := s.st.Query(ctx, RoleEvent)
	if since != "" {
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

func (s *CustodyStore) GetEvent(ctx context.Context, id int64) (models.Entry, error) {
	var out models.Entry
	err := s.st.Take(s.st.Query(ctx, RoleEvent).Where("id = ?", id), RoleEvent, &out, models.ErrNotFound)
	if err != nil {
		return models.Entry{}, classify(err)
	}
	return out, nil
}

var _ models.CustodyRepository = (*CustodyStore)(nil)
