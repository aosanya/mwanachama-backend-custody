package mwanachamacustody_test

import (
	"context"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func TestWithdrawnConsentsListsOnlyTheActorsWhoseConsentIsWithdrawnNow(t *testing.T) {
	db, s := newTestDB(t)
	clock := monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC))
	m, err := mwanachamacustody.NewCustodyManager(db, s, clock, nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	ctx := context.Background()
	version, err := m.CreateConsentVersion(ctx, mwanachamacustody.ConsentVersionInput{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateConsentVersion: %v", err)
	}
	if _, err := m.PublishConsentVersion(ctx, version.ID, "admin-1"); err != nil {
		t.Fatalf("PublishConsentVersion: %v", err)
	}
	for _, who := range []string{"gone", "back", "live"} {
		if _, err := m.GiveConsent(ctx, mwanachamacustody.GiveConsentInput{ActorID: who, StructureID: "ward-1"}); err != nil {
			t.Fatalf("GiveConsent %s: %v", who, err)
		}
	}
	gone, err := m.WithdrawConsent(ctx, "gone")
	if err != nil {
		t.Fatalf("WithdrawConsent gone: %v", err)
	}
	if _, err := m.WithdrawConsent(ctx, "back"); err != nil {
		t.Fatalf("WithdrawConsent back: %v", err)
	}
	if _, err := m.GiveConsent(ctx, mwanachamacustody.GiveConsentInput{ActorID: "back", StructureID: "ward-1"}); err != nil {
		t.Fatalf("GiveConsent back again: %v", err)
	}

	got, err := m.WithdrawnConsents(ctx)
	if err != nil {
		t.Fatalf("WithdrawnConsents: %v", err)
	}

	if len(got) != 1 || got[0] != gone {
		t.Fatalf("WithdrawnConsents = %+v, want only %+v", got, gone)
	}
}
