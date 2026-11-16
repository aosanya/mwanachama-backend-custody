package mwanachamacustody_test

import (
	"context"
	"sync"
	"testing"
	"time"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

func TestConsentTextPublishesFromSixManagersOnOneDatabaseNeverRunOutOfAttempts(t *testing.T) {
	const managerCount = 6
	const senders = 24
	const rounds = 15
	for round := 0; round < rounds; round++ {
		db := publishBackends[0].open(t)
		s, err := mwanachamacustody.LoadSpec(testSpec)
		if err != nil {
			t.Fatalf("LoadSpec: %v", err)
		}
		if err := mwanachamacustody.Provision(db, s); err != nil {
			t.Fatalf("Provision: %v", err)
		}
		managers := make([]*mwanachamacustody.CustodyManager, managerCount)
		for i := range managers {
			managers[i], err = mwanachamacustody.NewCustodyManager(db, s, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)), nil)
			if err != nil {
				t.Fatalf("NewCustodyManager: %v", err)
			}
		}

		start := make(chan struct{})
		var wg sync.WaitGroup
		errs := make([]error, senders)
		for i := 0; i < senders; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				<-start
				_, errs[i] = managers[i%managerCount].PublishConsentText(context.Background(), mwanachamacustody.ConsentTextInput{Text: "x"}, "staff-1")
			}(i)
		}
		close(start)
		wg.Wait()

		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d sender %d: %v", round, i, err)
			}
		}
		rows, err := managers[0].ListConsentVersions(context.Background(), "", "")
		if err != nil {
			t.Fatalf("ListConsentVersions: %v", err)
		}
		inForce := 0
		for _, row := range rows {
			if row.InForce {
				inForce++
			}
		}
		if len(rows) != senders || inForce != 1 {
			t.Fatalf("round %d rows = %d, in force = %d, want %d and 1", round, len(rows), inForce, senders)
		}
	}
}
