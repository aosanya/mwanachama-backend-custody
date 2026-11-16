package mwanachamacustody_test

import (
	"context"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func newAgreeWalk(t *testing.T) (*mwanachamacustody.CustodyManager, context.Context, string) {
	t.Helper()
	db, s := newTestDB(t)
	clock := monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	m, err := mwanachamacustody.NewCustodyManager(db, s, clock, nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	ctx := context.Background()
	version := publishMechanics(t, m, ctx, "v1")
	return m, ctx, version
}

func publishMechanics(t *testing.T, m *mwanachamacustody.CustodyManager, ctx context.Context, label string) string {
	t.Helper()
	version, err := m.CreateConsentVersion(ctx, mwanachamacustody.ConsentVersionInput{Scope: models.ScopeMechanics, Version: label, Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateConsentVersion %s: %v", label, err)
	}
	if _, err := m.PublishConsentVersion(ctx, version.ID, "admin-1"); err != nil {
		t.Fatalf("PublishConsentVersion %s: %v", label, err)
	}
	return version.ID
}

func giveConsent(t *testing.T, m *mwanachamacustody.CustodyManager, ctx context.Context, actor string) models.Record {
	t.Helper()
	rec, err := m.GiveConsent(ctx, mwanachamacustody.GiveConsentInput{ActorID: actor, StructureID: "ward-1"})
	if err != nil {
		t.Fatalf("GiveConsent %s: %v", actor, err)
	}
	return rec
}

func TestAgreeingTwiceToTheSameVersionAnswersTheFirstRecord(t *testing.T) {
	m, ctx, _ := newAgreeWalk(t)

	first := giveConsent(t, m, ctx, "resident-1")
	second := giveConsent(t, m, ctx, "resident-1")

	if second.ID != first.ID {
		t.Fatalf("second agree recorded %s, want the first record %s", second.ID, first.ID)
	}
	history, err := m.Consent().ListForActor(ctx, "resident-1")
	if err != nil {
		t.Fatalf("ListForActor: %v", err)
	}
	if len(history) != 1 {
		t.Fatalf("consent history holds %d records, want 1", len(history))
	}
}

func TestAgreeingAgainAfterWithdrawingRecordsANewConsent(t *testing.T) {
	m, ctx, _ := newAgreeWalk(t)
	first := giveConsent(t, m, ctx, "resident-1")
	if _, err := m.WithdrawConsent(ctx, "resident-1"); err != nil {
		t.Fatalf("WithdrawConsent: %v", err)
	}

	again := giveConsent(t, m, ctx, "resident-1")

	if again.ID == first.ID {
		t.Fatalf("agree after withdrawal answered the old record %s", first.ID)
	}
}

func TestAgreeingToANewVersionRecordsANewConsent(t *testing.T) {
	m, ctx, firstVersion := newAgreeWalk(t)
	first := giveConsent(t, m, ctx, "resident-1")
	secondVersion := publishMechanics(t, m, ctx, "v2")

	again := giveConsent(t, m, ctx, "resident-1")

	if again.ID == first.ID || again.MechanicsVersionID != secondVersion || secondVersion == firstVersion {
		t.Fatalf("agree to the new version answered %+v, want a new record for %s", again, secondVersion)
	}
}
