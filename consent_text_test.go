package mwanachamacustody_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func newTextManager(t *testing.T) *mwanachamacustody.CustodyManager {
	t.Helper()
	db, s := newTestDB(t)
	m, err := mwanachamacustody.NewCustodyManager(db, s, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)), nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	return m
}

func TestPublishConsentTextNumbersTheVersionsAndSupersedesTheLast(t *testing.T) {
	m := newTextManager(t)
	ctx := context.Background()
	first, err := m.PublishConsentText(ctx, mwanachamacustody.ConsentTextInput{Text: "  I agree.  "}, "staff-1")
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := m.PublishConsentText(ctx, mwanachamacustody.ConsentTextInput{Text: "I agree, again."}, "staff-1")
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if first.Version != "v1" || second.Version != "v2" {
		t.Errorf("versions = %q, %q, want v1, v2", first.Version, second.Version)
	}
	if first.PublishedBy != "" || second.PublishedBy != "" {
		t.Errorf("a published text names its publisher: %q, %q", first.PublishedBy, second.PublishedBy)
	}
	rows, err := m.ListConsentVersions(ctx, "", "")
	if err != nil {
		t.Fatalf("ListConsentVersions: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	inForce := 0
	for _, row := range rows {
		if row.PublishedBy != "" {
			t.Errorf("row %s names its publisher %q", row.Version, row.PublishedBy)
		}
		if row.InForce {
			inForce++
			if row.Version != "v2" {
				t.Errorf("in force = %s, want v2", row.Version)
			}
		}
	}
	if inForce != 1 {
		t.Errorf("rows in force = %d, want 1", inForce)
	}
	sheet, err := m.GetInForceConsent(ctx, "", "")
	if err != nil {
		t.Fatalf("GetInForceConsent: %v", err)
	}
	if sheet.ID != second.ID {
		t.Errorf("in force = %q, want %q", sheet.ID, second.ID)
	}
}

func TestPublishConsentTextRefusesWhatIsNotAText(t *testing.T) {
	m := newTextManager(t)
	ctx := context.Background()
	cases := map[string]mwanachamacustody.ConsentTextInput{
		"blank":         {Text: "   "},
		"too long":      {Text: strings.Repeat("x", mwanachamacustody.MaxConsentTextLength+1)},
		"unknown scope": {Text: "ok", Scope: "x"},
	}
	for name, in := range cases {
		if _, err := m.PublishConsentText(ctx, in, "staff-1"); !errors.Is(err, mwanachamacustody.ErrInvalidValue) {
			t.Errorf("%s = %v, want ErrInvalidValue", name, err)
		}
	}
	if _, err := m.GetInForceConsent(ctx, "x", ""); !errors.Is(err, mwanachamacustody.ErrInvalidValue) {
		t.Errorf("in force with an unknown scope = %v, want ErrInvalidValue", err)
	}
	if _, err := m.ListConsentVersions(ctx, "x", ""); !errors.Is(err, mwanachamacustody.ErrInvalidValue) {
		t.Errorf("list with an unknown scope = %v, want ErrInvalidValue", err)
	}
	if _, err := m.GetInForceConsent(ctx, string(models.ScopeAppendix), ""); !errors.Is(err, models.ErrNotFound) {
		t.Errorf("nothing published = %v, want ErrNotFound", err)
	}
}
