package mwanachamacustody_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"gorm.io/gorm"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

var errInjectedPublish = errors.New("injected publish failure")

func failNextUpdate(t *testing.T, db *gorm.DB) {
	t.Helper()
	var spent atomic.Bool
	err := db.Callback().Update().Before("gorm:update").Register("test:fail-once", func(tx *gorm.DB) {
		if spent.CompareAndSwap(false, true) {
			_ = tx.AddError(errInjectedPublish)
		}
	})
	if err != nil {
		t.Fatalf("register callback: %v", err)
	}
}

func TestFailedPublishLeavesNoDraftAndNoGapInTheNumbers(t *testing.T) {
	db, s := newTestDB(t)
	m, err := mwanachamacustody.NewCustodyManager(db, s, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)), nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	ctx := context.Background()
	failNextUpdate(t, db)

	if _, err := m.PublishConsentText(ctx, mwanachamacustody.ConsentTextInput{Text: "first"}, "staff-1"); !errors.Is(err, errInjectedPublish) {
		t.Fatalf("first publish = %v, want the injected failure", err)
	}
	rows, err := m.ListConsentVersions(ctx, "", "")
	if err != nil {
		t.Fatalf("ListConsentVersions: %v", err)
	}
	if len(rows) != 0 {
		t.Fatalf("rows after the failed publish = %d, want 0", len(rows))
	}

	published, err := m.PublishConsentText(ctx, mwanachamacustody.ConsentTextInput{Text: "second"}, "staff-1")
	if err != nil {
		t.Fatalf("second publish: %v", err)
	}
	if published.Version != "v1" {
		t.Fatalf("version = %q, want v1", published.Version)
	}
	rows, err = m.ListConsentVersions(ctx, "", "")
	if err != nil {
		t.Fatalf("ListConsentVersions: %v", err)
	}
	if len(rows) != 1 || !rows[0].InForce {
		t.Fatalf("rows = %+v, want one row in force", rows)
	}
}
