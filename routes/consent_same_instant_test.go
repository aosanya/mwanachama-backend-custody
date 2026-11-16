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
)

func managerWithClock(t *testing.T, clock mwanachamacustody.Clock) *mwanachamacustody.CustodyManager {
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
	cm, err := mwanachamacustody.NewCustodyManager(db, s, clock, nil)
	if err != nil {
		t.Fatalf("NewCustodyManager: %v", err)
	}
	return cm
}

func steppedBy(step time.Duration) mwanachamacustody.Clock {
	now := time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)
	return func() time.Time {
		at := now
		now = now.Add(step)
		return at
	}
}

func frozenAt(at time.Time) mwanachamacustody.Clock {
	return func() time.Time { return at }
}

func agreeAfterWithdraw(t *testing.T, cm *mwanachamacustody.CustodyManager) error {
	t.Helper()
	publishMechanics(t, cm)
	srv := serve(t, cm, "resident-1")
	if status, out := postJSON(t, srv.URL+"/consent/agree", `{}`); status != http.StatusCreated {
		t.Fatalf("first agree = %d (%v)", status, out)
	}
	if status, out := postJSON(t, srv.URL+"/consent/withdraw", `{}`); status != http.StatusCreated {
		t.Fatalf("withdraw = %d (%v)", status, out)
	}
	if status, out := postJSON(t, srv.URL+"/consent/agree", `{}`); status != http.StatusCreated {
		t.Fatalf("second agree = %d (%v)", status, out)
	}
	return cm.ConsentGate().Check(context.Background(), "resident-1")
}

func TestAgreeAfterWithdrawalWinsWhenTheClockIsFrozen(t *testing.T) {
	cm := managerWithClock(t, frozenAt(time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)))
	if err := agreeAfterWithdraw(t, cm); err != nil {
		t.Fatalf("Check = %v, want nil", err)
	}
}

func TestAgreeAfterWithdrawalWinsOneNanosecondLater(t *testing.T) {
	cm := managerWithClock(t, steppedBy(time.Nanosecond))
	if err := agreeAfterWithdraw(t, cm); err != nil {
		t.Fatalf("Check = %v, want nil", err)
	}
}

func TestAgreeAfterWithdrawalWinsWithinTheSameSecond(t *testing.T) {
	cm := managerWithClock(t, steppedBy(100*time.Millisecond))
	if err := agreeAfterWithdraw(t, cm); err != nil {
		t.Fatalf("Check = %v, want nil", err)
	}
}

func TestWithdrawalAfterAgreeStaysWithdrawnWhenTheClockIsFrozen(t *testing.T) {
	cm := managerWithClock(t, frozenAt(time.Date(2026, 3, 1, 9, 0, 0, 0, time.UTC)))
	publishMechanics(t, cm)
	srv := serve(t, cm, "resident-1")
	if status, out := postJSON(t, srv.URL+"/consent/agree", `{}`); status != http.StatusCreated {
		t.Fatalf("agree = %d (%v)", status, out)
	}
	if status, out := postJSON(t, srv.URL+"/consent/withdraw", `{}`); status != http.StatusCreated {
		t.Fatalf("withdraw = %d (%v)", status, out)
	}
	if err := cm.ConsentGate().Check(context.Background(), "resident-1"); !errors.Is(err, mwanachamacustody.ErrConsentWithdrawn) {
		t.Fatalf("Check = %v, want ErrConsentWithdrawn", err)
	}
}
