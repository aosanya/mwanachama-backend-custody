package mwanachamacustody_test

import (
	"context"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func TestConsentSheetInForceDoesNotNameWhoPublishedIt(t *testing.T) {
	db, s := newTestDB(t)
	m, err := mwanachamacustody.NewCustodyManager(db, s, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)), nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	ctx := context.Background()
	version, err := m.CreateConsentVersion(ctx, mwanachamacustody.ConsentVersionInput{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateConsentVersion: %v", err)
	}
	published, err := m.PublishConsentVersion(ctx, version.ID, "staff-1")
	if err != nil {
		t.Fatalf("PublishConsentVersion: %v", err)
	}
	if published.PublishedBy != "staff-1" {
		t.Fatalf("the publisher is not recorded: %q", published.PublishedBy)
	}

	sheet, err := m.GetConsentVersionInForce(ctx, string(models.ScopeMechanics), string(models.LanguageEnglish))
	if err != nil {
		t.Fatalf("GetConsentVersionInForce: %v", err)
	}

	if sheet.ID != version.ID {
		t.Errorf("in force = %q, want %q", sheet.ID, version.ID)
	}
	if sheet.PublishedBy != "" {
		t.Errorf("the sheet in force names its publisher %q, a staff account id a resident must not read", sheet.PublishedBy)
	}
}
