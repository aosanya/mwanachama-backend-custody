package routes_test

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func publishMechanics(t *testing.T, cm *mwanachamacustody.CustodyManager) {
	t.Helper()
	ctx := context.Background()
	v, err := cm.Consent().CreateVersion(ctx, models.TextVersion{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if _, err := cm.Consent().PublishVersion(ctx, v.ID, "admin-1", models.Now()); err != nil {
		t.Fatalf("PublishVersion: %v", err)
	}
}

func TestGiveRouteRecordsTheCallerConsent(t *testing.T) {
	cm, _ := newManager(t, nil)
	publishMechanics(t, cm)
	ctx := context.Background()
	if err := cm.ConsentGate().Check(ctx, "resident-1"); !errors.Is(err, mwanachamacustody.ErrConsentRequired) {
		t.Fatalf("Check before agreeing = %v, want ErrConsentRequired", err)
	}
	srv := serve(t, cm, "resident-1")
	status, out := postJSON(t, srv.URL+"/consent/agree", `{"actor_id":"someone-else"}`)
	if status != http.StatusCreated {
		t.Fatalf("agree status = %d (%v)", status, out)
	}
	if out["actor_id"] != "resident-1" {
		t.Fatalf("agreed actor = %v, want the caller", out["actor_id"])
	}
	if out["language"] != "en" {
		t.Fatalf("language = %v, want en when left out", out["language"])
	}
	if err := cm.ConsentGate().Check(ctx, "resident-1"); err != nil {
		t.Fatalf("Check after agreeing = %v", err)
	}
	if err := cm.ConsentGate().Check(ctx, "someone-else"); !errors.Is(err, mwanachamacustody.ErrConsentRequired) {
		t.Fatalf("other actor Check = %v, want ErrConsentRequired", err)
	}
}

func steppingManager(t *testing.T) *mwanachamacustody.CustodyManager {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("gorm.Open: %v", err)
	}
	s, err := mwanachamacustody.LoadSpec(testSpec)
	if err != nil {
		t.Fatalf("LoadSpec: %v", err)
	}
	if err := mwanachamacustody.Provision(db, s); err != nil {
		t.Fatalf("Provision: %v", err)
	}
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	clock := func() time.Time {
		now = now.Add(time.Minute)
		return now
	}
	cm, err := mwanachamacustody.NewCustodyManager(db, s, clock, nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	return cm
}

func TestGiveRouteRestoresConsentAfterWithdrawal(t *testing.T) {
	cm := steppingManager(t)
	publishMechanics(t, cm)
	ctx := context.Background()
	srv := serve(t, cm, "resident-1")
	if status, out := postJSON(t, srv.URL+"/consent/agree", `{}`); status != http.StatusCreated {
		t.Fatalf("first agree = %d (%v)", status, out)
	}
	if status, out := postJSON(t, srv.URL+"/consent/withdraw", `{}`); status != http.StatusCreated {
		t.Fatalf("withdraw = %d (%v)", status, out)
	}
	if err := cm.ConsentGate().Check(ctx, "resident-1"); !errors.Is(err, mwanachamacustody.ErrConsentWithdrawn) {
		t.Fatalf("Check after withdrawal = %v, want ErrConsentWithdrawn", err)
	}
	if status, out := postJSON(t, srv.URL+"/consent/agree", `{}`); status != http.StatusCreated {
		t.Fatalf("second agree = %d (%v)", status, out)
	}
	if err := cm.ConsentGate().Check(ctx, "resident-1"); err != nil {
		t.Fatalf("Check after agreeing again = %v", err)
	}
}

func TestGiveRouteIsAConflictWhileNoSheetIsInForce(t *testing.T) {
	cm, _ := newManager(t, nil)
	srv := serve(t, cm, "resident-1")
	status, out := postJSON(t, srv.URL+"/consent/agree", `{}`)
	if status != http.StatusConflict {
		t.Fatalf("agree with no sheet = %d (%v), want 409", status, out)
	}
}
