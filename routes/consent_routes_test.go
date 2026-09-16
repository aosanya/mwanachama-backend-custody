package routes_test

// Real-mux HTTP tests for routes.ConsentRoutes — this package had zero test
// coverage before this file (see documentation/3. implementation/todo.md's
// DEV-1695/DEV-1696).

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/routes"
)

type testIdentity struct{ id string }

func (i testIdentity) CallerID(r *http.Request) string { return i.id }

func newConsentTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	tables := mwanachamacustody.DefaultTableNames()
	if err := mwanachamacustody.Migrate(db, tables); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	consent, err := mwanachamacustody.NewConsentStore(db, tables)
	if err != nil {
		t.Fatalf("NewConsentStore: %v", err)
	}
	mux := http.NewServeMux()
	for _, rt := range routes.ConsentRoutes(consent, testIdentity{id: "operator-1"}) {
		mux.Handle(rt.Pattern(""), rt.Handler)
	}
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)
	return srv
}

func postJSON(t *testing.T, url, body string) (int, map[string]any) {
	t.Helper()
	resp, err := http.Post(url, "application/json", strings.NewReader(body))
	if err != nil {
		t.Fatalf("POST %s: %v", url, err)
	}
	defer resp.Body.Close()
	var out map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp.StatusCode, out
}

// TestCreateConsentVersion_HonorsCallerSuppliedID pins DEV-1695: POST
// /consent/versions accepts a caller-supplied "id" and stores it verbatim,
// letting any caller choose (and squat) the primary key of a
// consent_text_version row instead of the server minting one. This is the
// same caller-supplied-server-owned-field class the fleet already fixed
// elsewhere (mwanachama-backend-agency's AG21/WK17: "every Create<Type>
// method now owns ID/CreatedAt"), just not applied here.
//
// Once DEV-1695 is fixed (ConsentStore.CreateVersion clearing v.ID before
// insert, the way it already clears PublishedAt/PublishedBy/SupersededAt/
// CopiedMechanicsID), this assertion inverts: the returned id will be a
// server-minted "consentversion-..." id, not the literal string this test
// sends, and the test should be rewritten to assert the caller-supplied id
// is ignored.
func TestCreateConsentVersion_HonorsCallerSuppliedID(t *testing.T) {
	srv := newConsentTestServer(t)

	status, out := postJSON(t, srv.URL+"/consent/versions",
		`{"id":"attacker-chosen-id","scope":"mechanics","version":"v1","language":"en"}`)

	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %+v", status, out)
	}
	// Pins the current (broken) behavior: the caller's own literal id comes
	// straight back, proving the server never minted its own.
	if out["id"] != "attacker-chosen-id" {
		t.Fatalf("DEV-1695 appears fixed (server now controls id) — got %+v; update this test to assert the caller-supplied id was ignored instead", out)
	}
}

// TestCreateConsentVersion_DuplicateNaturalKeyReturns500NotConflict pins
// DEV-1696: routes.writeConsentErr's switch only maps
// mwanachamacustody.ErrNotFound/ErrAlreadyPublished — a genuine
// mwanachamacustody.ErrConflict (the store's own documented behavior for a
// second row on the same (scope, version, language) triple) falls through
// to the default branch and comes back as a bare 500 "internal error"
// instead of 409, hiding a normal, expected refusal behind an opaque
// server-error status.
//
// Once DEV-1696 is fixed (writeConsentErr gaining an
// errors.Is(err, mwanachamacustody.ErrConflict) case mapping to 409), this
// assertion inverts: expect 409, not 500.
func TestCreateConsentVersion_DuplicateNaturalKeyReturns500NotConflict(t *testing.T) {
	srv := newConsentTestServer(t)

	body := `{"scope":"mechanics","version":"v1","language":"en"}`
	if status, out := postJSON(t, srv.URL+"/consent/versions", body); status != http.StatusCreated {
		t.Fatalf("first create: status = %d, want 201: %+v", status, out)
	}

	status, out := postJSON(t, srv.URL+"/consent/versions", body)
	if status != http.StatusInternalServerError {
		t.Fatalf("DEV-1696 appears fixed (duplicate now maps to a real status) — got %d %+v; update this test to assert 409 instead", status, out)
	}
	if out["error"] != "internal error" {
		t.Fatalf("expected the generic masked message, got %+v", out)
	}
}
