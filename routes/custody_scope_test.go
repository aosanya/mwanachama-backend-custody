package routes_test

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

type emptySubtreeResolver struct{}

func (emptySubtreeResolver) Subtree(ctx context.Context, structureID string) ([]string, error) {
	return []string{}, nil
}

func TestListStructureActs_EmptySubtreeScopeLeaksOtherStructures(t *testing.T) {
	cm, _ := newManager(t, emptySubtreeResolver{})
	store := cm.Custody()

	ctx := context.Background()
	if _, err := store.AppendAct(ctx, mwanachamacustody.StructureActLogEntry{
		StructureID: "structure-A",
		Kind:        mwanachamacustody.ActRoleGranted,
		SubjectRef:  "subject-A",
		ActorLabel:  "actor A",
	}); err != nil {
		t.Fatalf("seed A: %v", err)
	}
	if _, err := store.AppendAct(ctx, mwanachamacustody.StructureActLogEntry{
		StructureID: "structure-B-SECRET",
		Kind:        mwanachamacustody.ActRoleGranted,
		SubjectRef:  "subject-B",
		ActorLabel:  "actor B",
	}); err != nil {
		t.Fatalf("seed B: %v", err)
	}

	srv := serve(t, cm, "operator-1")

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
