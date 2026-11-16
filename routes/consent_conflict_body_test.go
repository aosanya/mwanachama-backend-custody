package routes_test

import (
	"net/http"
	"regexp"
	"strings"
	"testing"
)

var driverCode = regexp.MustCompile(`\(\d+\)`)

func conflictMessage(t *testing.T, out map[string]any) string {
	t.Helper()
	message, _ := out["error"].(string)
	if strings.Contains(message, "mwanachama_") || driverCode.MatchString(message) {
		t.Fatalf("the 409 body shows the table name or a driver code: %q", message)
	}
	return message
}

func TestDuplicateNaturalKeyConflictNamesTheFieldsAndNothingPhysical(t *testing.T) {
	srv := newConsentTestServer(t)
	body := `{"scope":"mechanics","version":"v1","language":"en"}`
	if status, out := postJSON(t, srv.URL+"/consent/versions", body); status != http.StatusCreated {
		t.Fatalf("first create: %d %+v", status, out)
	}

	status, out := postJSON(t, srv.URL+"/consent/versions", body)

	if status != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %+v", status, out)
	}
	message := conflictMessage(t, out)
	for _, field := range []string{"scope", "version", "language"} {
		if !strings.Contains(message, field) {
			t.Errorf("message %q does not name %s", message, field)
		}
	}
}

func TestDuplicateIdConflictNamesTheIdAndNothingPhysical(t *testing.T) {
	srv := newConsentTestServer(t)
	if status, out := postJSON(t, srv.URL+"/consent/versions", `{"id":"same","scope":"mechanics","version":"v1","language":"en"}`); status != http.StatusCreated {
		t.Fatalf("first create: %d %+v", status, out)
	}

	status, out := postJSON(t, srv.URL+"/consent/versions", `{"id":"same","scope":"mechanics","version":"v2","language":"en"}`)

	if status != http.StatusConflict {
		t.Fatalf("status = %d, want 409: %+v", status, out)
	}
	if message := conflictMessage(t, out); !strings.Contains(message, "id") {
		t.Errorf("message %q does not name id", message)
	}
}
