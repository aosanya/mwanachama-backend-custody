package routes_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func TestWithdrawRouteTakesTheCallerConsentAway(t *testing.T) {
	cm, _ := newManager(t, nil)
	ctx := context.Background()
	v, err := cm.Consent().CreateVersion(ctx, models.TextVersion{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if _, err := cm.Consent().PublishVersion(ctx, v.ID, "admin-1", models.FormatTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("PublishVersion: %v", err)
	}
	if _, err := mwanachamacustody.Enroll(ctx, cm.Consent(), "operator-1", "ward-1", models.LanguageEnglish, models.FormatTime(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("Enroll: %v", err)
	}
	if err := cm.ConsentGate().Check(ctx, "operator-1"); err != nil {
		t.Fatalf("Check before withdrawal = %v", err)
	}
	srv := serve(t, cm, "operator-1")
	status, out := postJSON(t, srv.URL+"/consent/withdraw", `{"actor_id":"someone-else"}`)
	if status != http.StatusCreated {
		t.Fatalf("withdraw status = %d (%v)", status, out)
	}
	if out["actor_id"] != "operator-1" {
		t.Fatalf("withdrawn actor = %v, want the caller", out["actor_id"])
	}
	if err := cm.ConsentGate().Check(ctx, "operator-1"); !errors.Is(err, mwanachamacustody.ErrConsentWithdrawn) {
		t.Fatalf("Check after withdrawal = %v, want ErrConsentWithdrawn", err)
	}
	if err := cm.ConsentGate().Check(ctx, "someone-else"); !errors.Is(err, mwanachamacustody.ErrConsentRequired) {
		t.Fatalf("other actor Check = %v, want ErrConsentRequired", err)
	}
}
