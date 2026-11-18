package routes_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aosanya/mwanachama-backend-custody/routes"
)

func TestPaperConsentRouteRecordsConsentForTheNamedPerson(t *testing.T) {
	cm, _ := newManager(t, nil)
	publishMechanics(t, cm)
	srv := serve(t, cm, "clerk-1")
	status, out := postJSON(t, srv.URL+"/consent/recorded", `{"actor_id":"resident-9","channel":"paper"}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d (%v)", status, out)
	}
	if out["actor_id"] != "resident-9" || out["channel"] != "paper" || out["recorded_by"] != "clerk-1" {
		t.Fatalf("record = %v, want resident-9 on paper recorded by clerk-1", out)
	}
	if err := cm.ConsentGate().Check(context.Background(), "resident-9"); err != nil {
		t.Fatalf("Check after the clerk recorded consent = %v", err)
	}
	if err := cm.ConsentGate().Check(context.Background(), "clerk-1"); err == nil {
		t.Fatal("the clerk's own consent was recorded as well")
	}
}

func TestPaperConsentRouteRepeatedAnswersTheStandingRecord(t *testing.T) {
	cm, _ := newManager(t, nil)
	publishMechanics(t, cm)
	srv := serve(t, cm, "clerk-1")
	_, first := postJSON(t, srv.URL+"/consent/recorded", `{"actor_id":"resident-9","channel":"in_person"}`)
	status, second := postJSON(t, srv.URL+"/consent/recorded", `{"actor_id":"resident-9","channel":"paper"}`)
	if status != http.StatusCreated {
		t.Fatalf("second status = %d (%v)", status, second)
	}
	if first["id"] == nil || first["id"] != second["id"] {
		t.Fatalf("second record id = %v, want the first %v", second["id"], first["id"])
	}
	records, err := cm.Consent().ListForActor(context.Background(), "resident-9")
	if err != nil || len(records) != 1 {
		t.Fatalf("records = %d (%v), want 1", len(records), err)
	}
	if records[0].Channel != "in_person" {
		t.Fatalf("channel = %q, want the first channel kept", records[0].Channel)
	}
}

func TestPaperConsentRouteRefusesAChannelThatIsNotPaperOrInPerson(t *testing.T) {
	cm, _ := newManager(t, nil)
	publishMechanics(t, cm)
	srv := serve(t, cm, "clerk-1")
	for _, body := range []string{
		`{"actor_id":"resident-9","channel":"phone"}`,
		`{"actor_id":"resident-9"}`,
	} {
		status, out := postJSON(t, srv.URL+"/consent/recorded", body)
		if status != http.StatusBadRequest {
			t.Fatalf("%s: status = %d (%v), want 400", body, status, out)
		}
	}
	if err := cm.ConsentGate().Check(context.Background(), "resident-9"); err == nil {
		t.Fatal("a refused request still recorded consent")
	}
}

func TestPaperConsentRouteRefusesWithoutAPersonOrACaller(t *testing.T) {
	cm, _ := newManager(t, nil)
	publishMechanics(t, cm)
	status, out := postJSON(t, serve(t, cm, "clerk-1").URL+"/consent/recorded", `{"channel":"paper"}`)
	if status != http.StatusBadRequest {
		t.Fatalf("no person: status = %d (%v), want 400", status, out)
	}
	status, out = postJSON(t, serve(t, cm, "").URL+"/consent/recorded", `{"actor_id":"resident-9","channel":"paper"}`)
	if status < 400 || status >= 500 {
		t.Fatalf("no caller: status = %d (%v), want a 4xx", status, out)
	}
}

func TestPaperConsentRouteConflictsWhileNoMechanicsIsInForce(t *testing.T) {
	cm, _ := newManager(t, nil)
	status, out := postJSON(t, serve(t, cm, "clerk-1").URL+"/consent/recorded", `{"actor_id":"resident-9","channel":"paper"}`)
	if status != http.StatusConflict {
		t.Fatalf("status = %d (%v), want 409", status, out)
	}
}

func TestPaperConsentIsRecordedAfterAWithdrawal(t *testing.T) {
	cm, _ := newManager(t, nil)
	publishMechanics(t, cm)
	ctx := context.Background()
	srv := serve(t, cm, "clerk-1")
	postJSON(t, srv.URL+"/consent/recorded", `{"actor_id":"resident-9","channel":"paper"}`)
	if _, err := cm.WithdrawConsent(ctx, "resident-9"); err != nil {
		t.Fatalf("WithdrawConsent: %v", err)
	}
	status, out := postJSON(t, srv.URL+"/consent/recorded", `{"actor_id":"resident-9","channel":"paper"}`)
	if status != http.StatusCreated {
		t.Fatalf("status = %d (%v)", status, out)
	}
	if err := cm.ConsentGate().Check(ctx, "resident-9"); err != nil {
		t.Fatalf("Check after re-recording = %v", err)
	}
}

func TestPaperConsentRouteAsksForTheRecordActionAndRefusesWithout(t *testing.T) {
	cm, _ := newManager(t, nil)
	publishMechanics(t, cm)
	var asked []string
	refuse := func(_ context.Context, action string) error {
		asked = append(asked, action)
		return errors.New("no")
	}
	mux := http.NewServeMux()
	for _, rt := range routes.RoutesFor(cm, routes.Mount{Caller: callerIs("clerk-1"), Authorize: refuse}) {
		mux.HandleFunc(rt.Pattern(""), rt.Handler)
	}
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, httptest.NewRequest(http.MethodPost, "/consent/recorded", strings.NewReader(`{"actor_id":"resident-9","channel":"paper"}`)))
	if rec.Code != http.StatusForbidden {
		t.Fatalf("answered %d %s", rec.Code, rec.Body.String())
	}
	if len(asked) != 1 || asked[0] != "custody.consent.record" {
		t.Fatalf("asked %v", asked)
	}
	if err := cm.ConsentGate().Check(context.Background(), "resident-9"); err == nil {
		t.Fatal("a refused call still recorded consent")
	}
}
