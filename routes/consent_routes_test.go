package routes_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newConsentTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	cm, _ := newManager(t, nil)
	return serve(t, cm, "operator-1")
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

func TestCreateConsentVersion_HonorsCallerSuppliedID(t *testing.T) {
	srv := newConsentTestServer(t)

	status, out := postJSON(t, srv.URL+"/consent/versions",
		`{"id":"attacker-chosen-id","scope":"mechanics","version":"v1","language":"en"}`)

	if status != http.StatusCreated {
		t.Fatalf("status = %d, want 201: %+v", status, out)
	}

	if out["id"] != "attacker-chosen-id" {
		t.Fatalf("DEV-1695 appears fixed (server now controls id) — got %+v; update this test to assert the caller-supplied id was ignored instead", out)
	}
}

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

func TestCreateConsentVersionRefusesUnknownScopeAndLanguageWith400(t *testing.T) {
	srv := newConsentTestServer(t)
	for name, body := range map[string]string{
		"unknown scope":    `{"scope":"x","version":"1","language":"en"}`,
		"unknown language": `{"scope":"mechanics","version":"1","language":"fr"}`,
		"empty body":       `{}`,
	} {
		if status, out := postJSON(t, srv.URL+"/consent/versions", body); status != http.StatusBadRequest {
			t.Errorf("%s = %d (%v), want 400", name, status, out)
		}
	}
}
