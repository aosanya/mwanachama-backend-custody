package mwanachamacustody_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

type fixedSubtree map[string][]string

func (f fixedSubtree) Subtree(_ context.Context, id string) ([]string, error) {
	return f[id], nil
}

func newCoverManager(t *testing.T, resolver mwanachamacustody.ScopeResolver) *mwanachamacustody.CustodyManager {
	t.Helper()
	db, sp := newTestDB(t)
	clock := monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	m, err := mwanachamacustody.NewCustodyManager(db, sp, clock, resolver)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	return m
}

func appendAct(t *testing.T, m *mwanachamacustody.CustodyManager, structure string, kind models.ActKind) {
	t.Helper()
	if _, err := m.Custody().AppendAct(context.Background(), models.StructureActLogEntry{StructureID: structure, Kind: kind}); err != nil {
		t.Fatalf("AppendAct: %v", err)
	}
}

func TestListStructureActsRefusals(t *testing.T) {
	m := newCoverManager(t, nil)
	ctx := context.Background()
	cases := []struct {
		name                         string
		structure, scope, class, lim string
		want                         error
	}{
		{"no structure", "", "", "", "", mwanachamacustody.ErrStructureIDRequired},
		{"unknown scope", "w1", "galaxy", "", "", mwanachamacustody.ErrUnknownScope},
		{"unknown class", "w1", "", "nonsense", "", mwanachamacustody.ErrUnknownActClass},
		{"zero limit", "w1", "", "", "0", mwanachamacustody.ErrInvalidLimit},
		{"negative limit", "w1", "", "", "-3", mwanachamacustody.ErrInvalidLimit},
		{"non numeric limit", "w1", "", "", "many", mwanachamacustody.ErrInvalidLimit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := m.ListStructureActs(ctx, c.structure, c.scope, c.class, c.lim)
			if !errors.Is(err, c.want) {
				t.Fatalf("got %v, want %v", err, c.want)
			}
		})
	}
}

func TestListStructureActsNamesTheClassInTheRefusal(t *testing.T) {
	m := newCoverManager(t, nil)
	_, err := m.ListStructureActs(context.Background(), "w1", "", "bogus_class", "")
	if err == nil || err.Error() != mwanachamacustody.ErrUnknownActClass.Error()+": bogus_class" {
		t.Fatalf("got %v", err)
	}
}

func TestListStructureActsStructureScopeSeesOnlyThatStructure(t *testing.T) {
	m := newCoverManager(t, fixedSubtree{"parent": {"parent", "child"}})
	appendAct(t, m, "parent", models.ActRoleGranted)
	appendAct(t, m, "child", models.ActRoleRevoked)
	appendAct(t, m, "other", models.ActRoleGranted)

	rows, err := m.ListStructureActs(context.Background(), "parent", "structure", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["structure_id"] != "parent" {
		t.Fatalf("got %v", rows)
	}
}

func TestListStructureActsSubtreeScopeIncludesDescendantsNewestFirst(t *testing.T) {
	m := newCoverManager(t, fixedSubtree{"parent": {"parent", "child"}})
	appendAct(t, m, "parent", models.ActRoleGranted)
	appendAct(t, m, "child", models.ActRoleRevoked)
	appendAct(t, m, "other", models.ActRoleGranted)

	rows, err := m.ListStructureActs(context.Background(), "parent", "subtree", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 {
		t.Fatalf("got %d rows: %v", len(rows), rows)
	}
	if rows[0]["structure_id"] != "child" || rows[1]["structure_id"] != "parent" {
		t.Fatalf("not newest first: %v", rows)
	}
	if rows[0]["act_class"] != string(models.ClassRoles) {
		t.Fatalf("class: %v", rows[0]["act_class"])
	}
}

func TestListStructureActsSubtreeWithoutResolverFallsBackToTheStructure(t *testing.T) {
	m := newCoverManager(t, nil)
	appendAct(t, m, "parent", models.ActRoleGranted)
	appendAct(t, m, "child", models.ActRoleGranted)

	rows, err := m.ListStructureActs(context.Background(), "parent", "subtree", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0]["structure_id"] != "parent" {
		t.Fatalf("got %v", rows)
	}
}

func TestListStructureActsFiltersByClassAndLimit(t *testing.T) {
	m := newCoverManager(t, nil)
	appendAct(t, m, "w1", models.ActRoleGranted)
	appendAct(t, m, "w1", models.ActRoleRevoked)
	appendAct(t, m, "w1", models.ActPostWithheld)

	roles, err := m.ListStructureActs(context.Background(), "w1", "", string(models.ClassRoles), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(roles) != 2 {
		t.Fatalf("class filter: got %d", len(roles))
	}
	limited, err := m.ListStructureActs(context.Background(), "w1", "", "", "1")
	if err != nil {
		t.Fatal(err)
	}
	if len(limited) != 1 || limited[0]["act_kind"] != string(models.ActPostWithheld) {
		t.Fatalf("limit keeps the newest: %v", limited)
	}
}

func TestListStructureActsHugeLimitIsCappedNotRefused(t *testing.T) {
	m := newCoverManager(t, nil)
	appendAct(t, m, "w1", models.ActRoleGranted)
	rows, err := m.ListStructureActs(context.Background(), "w1", "", "", "999999")
	if err != nil || len(rows) != 1 {
		t.Fatalf("got %v %v", rows, err)
	}
}

func TestCountStructureActsTotalsAndPerClassIncludingZeroes(t *testing.T) {
	m := newCoverManager(t, nil)
	appendAct(t, m, "w1", models.ActRoleGranted)
	appendAct(t, m, "w1", models.ActRoleRevoked)
	appendAct(t, m, "w1", models.ActPostWithheld)

	total, by, err := m.CountStructureActs(context.Background(), "w1", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || by[string(models.ClassRoles)] != 2 || by[string(models.ClassModeration)] != 1 {
		t.Fatalf("total %d by %v", total, by)
	}
	if n, ok := by[string(models.ClassMembership)]; !ok || n != 0 {
		t.Fatalf("a class with no acts is listed as zero: %v", by)
	}
}

func TestCountStructureActsRefusals(t *testing.T) {
	m := newCoverManager(t, nil)
	ctx := context.Background()
	if _, _, err := m.CountStructureActs(ctx, "", "", ""); !errors.Is(err, mwanachamacustody.ErrStructureIDRequired) {
		t.Fatalf("no structure: %v", err)
	}
	if _, _, err := m.CountStructureActs(ctx, "w1", "galaxy", ""); !errors.Is(err, mwanachamacustody.ErrUnknownScope) {
		t.Fatalf("scope: %v", err)
	}
	if _, _, err := m.CountStructureActs(ctx, "w1", "", "yesterday"); !errors.Is(err, mwanachamacustody.ErrInvalidSince) {
		t.Fatalf("since: %v", err)
	}
}

func TestCountStructureActsSinceExcludesEarlierActs(t *testing.T) {
	m := newCoverManager(t, nil)
	appendAct(t, m, "w1", models.ActRoleGranted)
	appendAct(t, m, "w1", models.ActRoleRevoked)

	rows, err := m.ListStructureActs(context.Background(), "w1", "", "", "")
	if err != nil || len(rows) != 2 {
		t.Fatalf("seed: %v %v", rows, err)
	}
	newest := rows[0]["occurred_at"].(string)
	total, _, err := m.CountStructureActs(context.Background(), "w1", "", newest)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 {
		t.Fatalf("got %d, want 1", total)
	}
	total, _, err = m.CountStructureActs(context.Background(), "w1", "", "2030-01-01T00:00:00Z")
	if err != nil || total != 0 {
		t.Fatalf("future since: %d %v", total, err)
	}
}

func TestCustodyEventsRefusalsAndListing(t *testing.T) {
	m := newCoverManager(t, nil)
	ctx := context.Background()
	if _, err := m.ListCustodyEvents(ctx, "nonsense", ""); !errors.Is(err, mwanachamacustody.ErrUnknownEventChip) {
		t.Fatalf("chip: %v", err)
	}
	if _, err := m.ListCustodyEvents(ctx, "", "0"); !errors.Is(err, mwanachamacustody.ErrInvalidLimit) {
		t.Fatalf("limit: %v", err)
	}
	if _, _, err := m.CountCustodyEvents(ctx, "not-a-time"); !errors.Is(err, mwanachamacustody.ErrInvalidSince) {
		t.Fatalf("since: %v", err)
	}

	chip, err := models.EventChipOf(models.EventExportStarted)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := m.Custody().AppendEvent(ctx, models.Entry{Kind: models.EventExportStarted, Detail: "run"}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := m.ListCustodyEvents(ctx, string(chip), "2")
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 2 || rows[0]["event_kind"] != string(models.EventExportStarted) || rows[0]["event_chip"] != string(chip) {
		t.Fatalf("got %v", rows)
	}
	if rows[0]["id"].(int64) < rows[1]["id"].(int64) {
		t.Fatalf("not newest first: %v", rows)
	}

	total, by, err := m.CountCustodyEvents(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 || by[string(chip)] != 3 {
		t.Fatalf("total %d by %v", total, by)
	}
}

func TestOwnRecordsAreReadableByTheirSubjectOnly(t *testing.T) {
	m := newCoverManager(t, nil)
	ctx := context.Background()
	if _, err := m.ListActorExports(ctx, "alice", "bob"); !errors.Is(err, mwanachamacustody.ErrNotSelf) {
		t.Fatalf("exports: %v", err)
	}
	if _, err := m.ListContactReads(ctx, "alice", "bob"); !errors.Is(err, mwanachamacustody.ErrNotSelf) {
		t.Fatalf("contact reads: %v", err)
	}
	if _, err := m.ListActorExports(ctx, "alice", "alice"); err != nil {
		t.Fatalf("own exports: %v", err)
	}
	if _, err := m.ListContactReads(ctx, "alice", "alice"); err != nil {
		t.Fatalf("own contact reads: %v", err)
	}
}

func TestExportJobAndConsentVersionLookupsAnswerNotFound(t *testing.T) {
	m := newCoverManager(t, nil)
	ctx := context.Background()
	if _, err := m.GetExportJob(ctx, "no-such-job"); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("export job: %v", err)
	}
	if _, err := m.GetConsentVersion(ctx, "no-such-version"); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("consent version: %v", err)
	}
	if _, err := m.GetConsentVersionInForce(ctx, string(models.ScopeMechanics), string(models.LanguageEnglish)); !errors.Is(err, models.ErrNotFound) {
		t.Fatalf("in force: %v", err)
	}
}
