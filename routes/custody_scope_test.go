package routes_test

// Real-mux HTTP test for routes.CustodyRoutes's ?scope=subtree handling —
// pins DEV-1697.

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/routes"
)

// emptySubtreeResolver always answers "no descendants" with no error — a
// plausible real answer for a leaf structure, or an unknown structure id,
// depending on how a mounting host's own resolver is written.
type emptySubtreeResolver struct{}

func (emptySubtreeResolver) Subtree(ctx context.Context, structureID string) ([]string, error) {
	return []string{}, nil
}

// TestListStructureActs_EmptySubtreeScopeLeaksOtherStructures pins DEV-1697:
// CustodyStore.ListActs/.CountActsByClass only add a `chapter_id IN (...)`
// filter `if len(structures) > 0` — a scope resolver that legitimately
// returns an empty, non-nil, no-error slice (a leaf structure with zero
// descendants, or any resolver written to return "found nothing" rather
// than an error) makes the query fall through to unscoped, returning every
// structure's act-log rows and counts, not zero of them.
//
// Once DEV-1697 is fixed (ListActs/CountActsByClass treating an
// empty-but-non-nil structures slice as "match nothing", the same way an IN
// () clause should), this test's assertions on entry_count/everything
// should both read 1 (structure-A's own row only, none of
// structure-B-SECRET's), and this comment should be updated accordingly.
func TestListStructureActs_EmptySubtreeScopeLeaksOtherStructures(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := mwanachamacustody.DefaultTableNames()
	if err := mwanachamacustody.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	custody, err := mwanachamacustody.NewCustodyStore(db, tables, mwanachamacustody.SystemClock)
	if err != nil {
		t.Fatalf("NewCustodyStore: %v", err)
	}

	ctx := context.Background()
	if _, err := custody.AppendAct(ctx, mwanachamacustody.StructureActLogEntry{
		StructureID: "structure-A",
		Kind:        mwanachamacustody.ActRoleGranted,
		SubjectRef:  "subject-A",
		ActorLabel:  "actor A",
	}); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if _, err := custody.AppendAct(ctx, mwanachamacustody.StructureActLogEntry{
		StructureID: "structure-B-SECRET",
		Kind:        mwanachamacustody.ActRoleGranted,
		SubjectRef:  "subject-B",
		ActorLabel:  "actor B",
	}); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	mux := http.NewServeMux()
	for _, rt := range routes.CustodyRoutes(custody, emptySubtreeResolver{}) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	defer srv.Close()

	// A caller who can only see structure-A asks for its subtree act-log.
	resp, err := http.Get(srv.URL + "/structures/structure-A/act-log?scope=subtree")
	if err != nil {
		t.Fatalf("GET act-log: %v", err)
	}
	defer resp.Body.Close()
	var out struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	// Pins the current (broken) behavior: both structures' entries come
	// back, including structure-B-SECRET's, which this caller never asked
	// about and whose id never appeared in the resolved scope.
	if len(out.Entries) != 2 {
		t.Fatalf("DEV-1697 appears fixed (empty subtree scope no longer leaks) — got %d entries, want 2 (the broken/leaking count); update this test to assert 1 entry (structure-A's own only)", len(out.Entries))
	}
	sawLeaked := false
	for _, e := range out.Entries {
		if e["structure_id"] == "structure-B-SECRET" {
			sawLeaked = true
		}
	}
	if !sawLeaked {
		t.Fatalf("expected structure-B-SECRET's row to leak through (pinning the bug); entries: %+v", out.Entries)
	}

	// Same leak on the /counts endpoint.
	resp2, err := http.Get(srv.URL + "/structures/structure-A/act-log/counts?scope=subtree")
	if err != nil {
		t.Fatalf("GET counts: %v", err)
	}
	defer resp2.Body.Close()
	var out2 struct {
		Everything int `json:"everything"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&out2); err != nil {
		t.Fatalf("decode counts: %v", err)
	}
	if out2.Everything != 2 {
		t.Fatalf("DEV-1697 appears fixed on the counts path too — got everything=%d, want 2 (the broken/leaking count); update this test to assert 1", out2.Everything)
	}
}
