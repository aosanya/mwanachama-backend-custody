package mwanachamacustody_test

import (
	"context"
	"errors"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
	"github.com/aosanya/mwanachama-backend-custody/models"
)

func gateFixture(t *testing.T, clock mwanachamacustody.Clock) (*mwanachamacustody.Consent, *mwanachamacustody.ConsentStore) {
	t.Helper()
	s := newConsentStore(t)
	ctx := context.Background()
	v, err := s.CreateVersion(ctx, models.TextVersion{Scope: models.ScopeMechanics, Version: "v1", Language: models.LanguageEnglish})
	if err != nil {
		t.Fatalf("CreateVersion: %v", err)
	}
	if _, err := s.PublishVersion(ctx, v.ID, "admin-1", models.FormatTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))); err != nil {
		t.Fatalf("PublishVersion: %v", err)
	}
	return mwanachamacustody.NewConsent(s, mwanachamacustody.NewMemoryWithdrawalLedger(), clock), s
}

func enrollAt(t *testing.T, s *mwanachamacustody.ConsentStore, actor string, at time.Time) {
	t.Helper()
	if _, err := mwanachamacustody.Enroll(context.Background(), s, actor, "ward-1", models.LanguageEnglish, models.FormatTime(at)); err != nil {
		t.Fatalf("Enroll: %v", err)
	}
}

func TestGateDeniesAnActorWithNoConsentRecord(t *testing.T) {
	c, _ := gateFixture(t, nil)
	if err := c.Check(context.Background(), "nobody"); !errors.Is(err, mwanachamacustody.ErrConsentRequired) {
		t.Fatalf("Check = %v, want ErrConsentRequired", err)
	}
}

func TestGateAllowsAnActorWhoConsented(t *testing.T) {
	c, s := gateFixture(t, nil)
	enrollAt(t, s, "mem-1", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	if err := c.Check(context.Background(), "mem-1"); err != nil {
		t.Fatalf("Check = %v, want nil", err)
	}
}

func TestGateDeniesAfterWithdrawal(t *testing.T) {
	c, s := gateFixture(t, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)))
	ctx := context.Background()
	enrollAt(t, s, "mem-1", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	if _, err := c.Withdraw(ctx, "mem-1"); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	if err := c.Check(ctx, "mem-1"); !errors.Is(err, mwanachamacustody.ErrConsentWithdrawn) {
		t.Fatalf("Check = %v, want ErrConsentWithdrawn", err)
	}
	if err := c.Check(ctx, "mem-2"); !errors.Is(err, mwanachamacustody.ErrConsentRequired) {
		t.Fatalf("other actor Check = %v, want ErrConsentRequired", err)
	}
}

func TestGateAllowsAgainAfterFreshConsent(t *testing.T) {
	c, s := gateFixture(t, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)))
	ctx := context.Background()
	enrollAt(t, s, "mem-1", time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	if _, err := c.Withdraw(ctx, "mem-1"); err != nil {
		t.Fatalf("Withdraw: %v", err)
	}
	enrollAt(t, s, "mem-1", time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC))
	if err := c.Check(ctx, "mem-1"); err != nil {
		t.Fatalf("Check after re-consent = %v, want nil", err)
	}
}

func TestWithdrawalReachesEverySubscriberAndReportsFailures(t *testing.T) {
	c, _ := gateFixture(t, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)))
	var got []string
	boom := errors.New("engage down")
	c.Subscribe(func(_ context.Context, w mwanachamacustody.Withdrawal) error {
		got = append(got, "comm:"+w.ActorID)
		return boom
	})
	c.Subscribe(func(_ context.Context, w mwanachamacustody.Withdrawal) error {
		got = append(got, "engage:"+w.ActorID)
		return nil
	})
	w, err := c.Withdraw(context.Background(), "mem-1")
	if !errors.Is(err, boom) {
		t.Fatalf("Withdraw err = %v, want the subscriber failure", err)
	}
	if w.ActorID != "mem-1" || w.WithdrawnAt == "" {
		t.Fatalf("Withdrawal = %+v", w)
	}
	if len(got) != 2 || got[0] != "comm:mem-1" || got[1] != "engage:mem-1" {
		t.Fatalf("subscribers saw %v", got)
	}
}

func TestGateHelperIsOpenWhenNoGateIsWired(t *testing.T) {
	if err := mwanachamacustody.Gate(context.Background(), nil, "mem-1"); err != nil {
		t.Fatalf("Gate(nil) = %v", err)
	}
	c, _ := gateFixture(t, nil)
	if err := mwanachamacustody.Gate(context.Background(), c, "mem-1"); !errors.Is(err, mwanachamacustody.ErrConsentRequired) {
		t.Fatalf("Gate = %v", err)
	}
}

func TestDBLedgerKeepsTheLatestWithdrawalAcrossInstances(t *testing.T) {
	db, _ := newTestDB(t)
	ctx := context.Background()
	first, err := mwanachamacustody.NewDBWithdrawalLedger(db, "test_withdrawal")
	if err != nil {
		t.Fatalf("NewDBWithdrawalLedger: %v", err)
	}
	for _, at := range []time.Time{time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC), time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)} {
		if err := first.Record(ctx, mwanachamacustody.Withdrawal{ActorID: "mem-1", WithdrawnAt: models.FormatTime(at)}); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	second, err := mwanachamacustody.NewDBWithdrawalLedger(db, "test_withdrawal")
	if err != nil {
		t.Fatalf("second NewDBWithdrawalLedger: %v", err)
	}
	got, ok, err := second.Latest(ctx, "mem-1")
	if err != nil || !ok || got.WithdrawnAt != models.FormatTime(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("Latest = %+v, %v, %v", got, ok, err)
	}
	if _, ok, _ := second.Latest(ctx, "mem-2"); ok {
		t.Fatalf("Latest for another actor found a withdrawal")
	}
	if _, err := mwanachamacustody.NewDBWithdrawalLedger(db, "bad name; drop"); err == nil {
		t.Fatalf("a table name with spaces was accepted")
	}
}
