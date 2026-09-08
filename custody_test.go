package mwanachamacustody_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func newCustodyStore(t *testing.T, clock mwanachamacustody.Clock) *mwanachamacustody.CustodyStore {
	t.Helper()
	db, tables := newTestDB(t)
	s, err := mwanachamacustody.NewCustodyStore(db, tables, clock)
	if err != nil {
		t.Fatalf("NewCustodyStore: %v", err)
	}
	return s
}

// TestActKindsAllHaveAClass is the census DEV-1257-style test that proves
// actClasses is total: every ActKind constant reaches a class, so
// ActClassOf never refuses a kind this package itself declares live.
func TestActKindsAllHaveAClass(t *testing.T) {
	for _, k := range []models.ActKind{
		models.ActPostWithheld, models.ActPostRestored, models.ActReportLeftStanding,
		models.ActRemovalLeftStanding, models.ActRoleGranted, models.ActRoleRevoked,
		models.ActCaseEscalated, models.ActTransferredOut, models.ActTransferredIn,
		models.ActLeftStructure, models.ActMembershipEnded, models.ActStructureRetired,
		models.ActAnswerReadNamed, models.ActCheckStarted,
	} {
		if _, err := models.ActClassOf(k); err != nil {
			t.Errorf("ActClassOf(%q): %v", k, err)
		}
	}
}

func TestAppendActRefusesUnknownKind(t *testing.T) {
	s := newCustodyStore(t, mwanachamacustody.SystemClock)
	_, err := s.AppendAct(context.Background(), models.StructureActLogEntry{
		StructureID: "ward-1", Kind: models.ActKind("no_such_kind"),
	})
	if !errors.Is(err, models.ErrUnknownActKind) {
		t.Fatalf("AppendAct with unknown kind: got %v, want ErrUnknownActKind", err)
	}
}

func TestAppendActRefusesEmptyStructure(t *testing.T) {
	s := newCustodyStore(t, mwanachamacustody.SystemClock)
	_, err := s.AppendAct(context.Background(), models.StructureActLogEntry{Kind: models.ActRoleGranted})
	if !errors.Is(err, models.ErrStructureRequired) {
		t.Fatalf("AppendAct with no structure: got %v, want ErrStructureRequired", err)
	}
}

func TestAppendActDerivesClassAndOrdersNewestFirst(t *testing.T) {
	clock := monotonicClock(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	s := newCustodyStore(t, clock)
	ctx := context.Background()

	first, err := s.AppendAct(ctx, models.StructureActLogEntry{StructureID: "ward-1", Kind: models.ActRoleGranted})
	if err != nil {
		t.Fatalf("AppendAct #1: %v", err)
	}
	if first.Class != models.ClassRoles {
		t.Fatalf("Class = %q, want %q", first.Class, models.ClassRoles)
	}
	second, err := s.AppendAct(ctx, models.StructureActLogEntry{StructureID: "ward-1", Kind: models.ActRoleRevoked})
	if err != nil {
		t.Fatalf("AppendAct #2: %v", err)
	}

	got, err := s.ListActs(ctx, []string{"ward-1"}, "", 0)
	if err != nil {
		t.Fatalf("ListActs: %v", err)
	}
	if len(got) != 2 || got[0].ID != second.ID || got[1].ID != first.ID {
		t.Fatalf("ListActs order = %+v, want [second, first]", got)
	}
}

func TestListActsScopesToNamedStructuresAndClass(t *testing.T) {
	s := newCustodyStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	mustAppendAct(t, s, "ward-1", models.ActRoleGranted)
	mustAppendAct(t, s, "ward-2", models.ActRoleGranted)
	mustAppendAct(t, s, "ward-1", models.ActStructureRetired)

	got, err := s.ListActs(ctx, []string{"ward-1"}, models.ClassRoles, 0)
	if err != nil {
		t.Fatalf("ListActs: %v", err)
	}
	if len(got) != 1 || got[0].StructureID != "ward-1" || got[0].Class != models.ClassRoles {
		t.Fatalf("ListActs = %+v, want one roles act at ward-1", got)
	}
}

func TestCountActsByClassSumsToEverything(t *testing.T) {
	s := newCustodyStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	mustAppendAct(t, s, "ward-1", models.ActRoleGranted)
	mustAppendAct(t, s, "ward-1", models.ActRoleRevoked)
	mustAppendAct(t, s, "ward-1", models.ActStructureRetired)

	counts, err := s.CountActsByClass(ctx, []string{"ward-1"}, time.Time{})
	if err != nil {
		t.Fatalf("CountActsByClass: %v", err)
	}
	total := 0
	for _, c := range models.ActClasses() {
		total += counts[c]
	}
	if total != 3 {
		t.Fatalf("sum of chips = %d, want 3 (counts=%v)", total, counts)
	}
}

func TestAppendEventRefusesUnknownKind(t *testing.T) {
	s := newCustodyStore(t, mwanachamacustody.SystemClock)
	_, err := s.AppendEvent(context.Background(), models.Entry{Kind: models.EventKind("no_such_kind"), ActorLabel: "x", Detail: "x"})
	if !errors.Is(err, models.ErrUnknownEventKind) {
		t.Fatalf("AppendEvent with unknown kind: got %v, want ErrUnknownEventKind", err)
	}
}

func TestAppendEventDerivesChipAndGetEventRoundTrips(t *testing.T) {
	s := newCustodyStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	e, err := s.AppendEvent(ctx, models.Entry{
		Kind: models.EventPhoneSaltRotated, ActorLabel: "system", Detail: "rotated",
		Evidence: map[string]any{"key_id": "7 -> 8"},
	})
	if err != nil {
		t.Fatalf("AppendEvent: %v", err)
	}
	if e.Chip != models.ChipKeys {
		t.Fatalf("Chip = %q, want %q", e.Chip, models.ChipKeys)
	}
	got, err := s.GetEvent(ctx, e.ID)
	if err != nil {
		t.Fatalf("GetEvent: %v", err)
	}
	if got.Evidence["key_id"] != "7 -> 8" {
		t.Fatalf("GetEvent Evidence = %v, want key_id round-tripped", got.Evidence)
	}
}

func TestGetEventUnknownIDReturnsErrNotFound(t *testing.T) {
	s := newCustodyStore(t, mwanachamacustody.SystemClock)
	_, err := s.GetEvent(context.Background(), 999999)
	if !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("GetEvent unknown id: got %v, want ErrNotFound", err)
	}
}

func TestCountEventsByChipSumsToEverything(t *testing.T) {
	s := newCustodyStore(t, mwanachamacustody.SystemClock)
	ctx := context.Background()
	for _, k := range []models.EventKind{models.EventPhoneSaltRotated, models.EventKeysRotated, models.EventThemesRecomputed} {
		if _, err := s.AppendEvent(ctx, models.Entry{Kind: k, ActorLabel: "system", Detail: "x"}); err != nil {
			t.Fatalf("AppendEvent(%q): %v", k, err)
		}
	}
	counts, err := s.CountEventsByChip(ctx, time.Time{})
	if err != nil {
		t.Fatalf("CountEventsByChip: %v", err)
	}
	total := 0
	for _, c := range models.EventChips() {
		total += counts[c]
	}
	if total != 3 {
		t.Fatalf("sum of chips = %d, want 3 (counts=%v)", total, counts)
	}
}

func mustAppendAct(t *testing.T, s *mwanachamacustody.CustodyStore, structureID string, kind models.ActKind) models.StructureActLogEntry {
	t.Helper()
	e, err := s.AppendAct(context.Background(), models.StructureActLogEntry{StructureID: structureID, Kind: kind})
	if err != nil {
		t.Fatalf("AppendAct(%q, %q): %v", structureID, kind, err)
	}
	return e
}
