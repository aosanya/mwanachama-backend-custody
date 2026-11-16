package mwanachamacustody_test

import (
	"context"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func TestPublishConsentTextSkipsALabelAHandMadeDraftAlreadyTook(t *testing.T) {
	db, s := newTestDB(t)
	m, err := mwanachamacustody.NewCustodyManager(db, s, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)), nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	ctx := context.Background()
	if _, err := m.CreateConsentVersion(ctx, mwanachamacustody.ConsentVersionInput{
		Scope:    models.ScopeMechanics,
		Version:  "v2",
		Language: models.LanguageEnglish,
		Clauses:  []models.Clause{{Key: "consent", Text: "a draft made by hand"}},
	}); err != nil {
		t.Fatalf("hand-made draft: %v", err)
	}

	published, err := m.PublishConsentText(ctx, mwanachamacustody.ConsentTextInput{Text: "second"}, "staff-1")
	if err != nil {
		t.Fatalf("publish with v2 taken: %v", err)
	}
	if published.Version != "v3" {
		t.Fatalf("version = %q, want v3", published.Version)
	}
	rows, err := m.ListConsentVersions(ctx, "", "")
	if err != nil {
		t.Fatalf("ListConsentVersions: %v", err)
	}
	inForce := ""
	for _, row := range rows {
		if row.InForce {
			inForce = row.Version
		}
	}
	if len(rows) != 2 || inForce != "v3" {
		t.Fatalf("rows = %d, in force = %q, want 2 and v3", len(rows), inForce)
	}
}
