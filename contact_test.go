package mwanachamacustody_test

import (
	"context"
	"errors"
	"testing"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func newContactStore(t *testing.T, clock mwanachamacustody.Clock) *mwanachamacustody.ContactStore {
	t.Helper()
	db, tables := newTestDB(t)
	s, err := mwanachamacustody.NewContactStore(db, tables, clock)
	if err != nil {
		t.Fatalf("NewContactStore: %v", err)
	}
	return s
}

func TestContactCreateIsEventNotCounter(t *testing.T) {
	s := newContactStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if _, err := s.Create(ctx, models.Read{ActorID: "mem-1", ReadBy: "op-1", StructureID: "ward-1"}); err != nil {
			t.Fatalf("Create #%d: %v", i, err)
		}
	}
	reads, err := s.ListForSubject(ctx, "mem-1")
	if err != nil {
		t.Fatalf("ListForSubject: %v", err)
	}
	if len(reads) != 2 {
		t.Fatalf("ListForSubject = %d rows, want 2 (event, not counter)", len(reads))
	}
}

func TestContactListForSubjectAndByOperatorAreDisjointScopes(t *testing.T) {
	s := newContactStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	mustCreateRead(t, s, "mem-1", "op-1", "ward-1")
	mustCreateRead(t, s, "mem-2", "op-1", "ward-1")
	mustCreateRead(t, s, "mem-1", "op-2", "ward-1")

	subj, err := s.ListForSubject(ctx, "mem-1")
	if err != nil {
		t.Fatalf("ListForSubject: %v", err)
	}
	if len(subj) != 2 {
		t.Fatalf("ListForSubject(mem-1) = %d, want 2 (both operators who read mem-1)", len(subj))
	}

	byOp, err := s.ListByOperator(ctx, "op-1")
	if err != nil {
		t.Fatalf("ListByOperator: %v", err)
	}
	if len(byOp) != 2 {
		t.Fatalf("ListByOperator(op-1) = %d, want 2 (both actors op-1 read)", len(byOp))
	}
}

func TestContactNotifiedReflectsNotifiedAt(t *testing.T) {
	s := newContactStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	r := mustCreateRead(t, s, "mem-1", "op-1", "ward-1")
	if r.Notified() {
		t.Fatalf("Notified() true before NotifiedAt set")
	}
	got, err := s.Get(ctx, r.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Notified() {
		t.Fatalf("Get().Notified() true, want false — no setter exists on this repository")
	}
}

func TestContactGetUnknownIDReturnsErrNotFound(t *testing.T) {
	s := newContactStore(t, mwanachamacustody.SystemClock)
	_, err := s.Get(context.Background(), "no-such-read")
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("Get unknown id: got %v, want ErrNotFound", err)
	}
}

func mustCreateRead(t *testing.T, s *mwanachamacustody.ContactStore, actorID, readBy, structureID string) models.Read {
	t.Helper()
	r, err := s.Create(context.Background(), models.Read{ActorID: actorID, ReadBy: readBy, StructureID: structureID})
	if err != nil {
		t.Fatalf("Create(%q, %q): %v", actorID, readBy, err)
	}
	return r
}
