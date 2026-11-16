package mwanachamacustody_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	mwanachamacustody "github.com/aosanya/mwanachama-backend-custody"
)

type publishBackend struct {
	name string
	open func(t *testing.T) *gorm.DB
}

var publishBackends = []publishBackend{
	{"sqlite file with several connections", func(t *testing.T) *gorm.DB {
		dsn := "file:" + filepath.Join(t.TempDir(), "custody.db") + "?_pragma=busy_timeout(10000)&_txlock=immediate"
		db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
		if err != nil {
			t.Fatalf("gorm.Open: %v", err)
		}
		return db
	}},
	{"in-memory sqlite on one connection", func(t *testing.T) *gorm.DB {
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			t.Fatalf("gorm.Open: %v", err)
		}
		sqlDB, err := db.DB()
		if err != nil {
			t.Fatalf("db.DB: %v", err)
		}
		sqlDB.SetMaxOpenConns(1)
		return db
	}},
}

func TestConsentTextPublishesAtOnceAreNumberedOneAfterTheOther(t *testing.T) {
	const senders = 8
	const rounds = 5
	for _, backend := range publishBackends {
		t.Run(backend.name, func(t *testing.T) {
			for round := 0; round < rounds; round++ {
				db := backend.open(t)
				s, err := mwanachamacustody.LoadSpec(testSpec)
				if err != nil {
					t.Fatalf("LoadSpec: %v", err)
				}
				if err := mwanachamacustody.Provision(db, s); err != nil {
					t.Fatalf("Provision: %v", err)
				}
				m, err := mwanachamacustody.NewCustodyManager(db, s, monotonicClock(time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)), nil)
				if err != nil {
					t.Fatalf("NewCustodyManager: %v", err)
				}

				start := make(chan struct{})
				var wg sync.WaitGroup
				errs := make([]error, senders)
				for i := 0; i < senders; i++ {
					wg.Add(1)
					go func(i int) {
						defer wg.Done()
						<-start
						_, errs[i] = m.PublishConsentText(context.Background(), mwanachamacustody.ConsentTextInput{Text: "x"}, "staff-1")
					}(i)
				}
				close(start)
				wg.Wait()

				for i, err := range errs {
					if err != nil {
						t.Fatalf("round %d sender %d: %v (conflict=%v)", round, i, err, errors.Is(err, mwanachamacustody.ErrConflict))
					}
				}
				rows, err := m.ListConsentVersions(context.Background(), "", "")
				if err != nil {
					t.Fatalf("ListConsentVersions: %v", err)
				}
				var labels []string
				var inForce []string
				for _, row := range rows {
					labels = append(labels, row.Version)
					if row.InForce {
						inForce = append(inForce, row.Version)
					}
				}
				sort.Strings(labels)
				want := make([]string, 0, senders)
				for i := 1; i <= senders; i++ {
					want = append(want, fmt.Sprintf("v%d", i))
				}
				sort.Strings(want)
				if fmt.Sprint(labels) != fmt.Sprint(want) {
					t.Fatalf("round %d versions = %v, want %v", round, labels, want)
				}
				if len(inForce) != 1 || inForce[0] != fmt.Sprintf("v%d", senders) {
					t.Fatalf("round %d in force = %v, want only v%d", round, inForce, senders)
				}
			}
		})
	}
}

func TestConsentTextPublishesFromTwoManagersOnOneDatabaseAreNumberedOneAfterTheOther(t *testing.T) {
	const senders = 8
	for round := 0; round < 5; round++ {
		db := publishBackends[0].open(t)
		s, err := mwanachamacustody.LoadSpec(testSpec)
		if err != nil {
			t.Fatalf("LoadSpec: %v", err)
		}
		if err := mwanachamacustody.Provision(db, s); err != nil {
			t.Fatalf("Provision: %v", err)
		}
		managers := make([]*mwanachamacustody.CustodyManager, 2)
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
				_, errs[i] = managers[i%2].PublishConsentText(context.Background(), mwanachamacustody.ConsentTextInput{Text: "x"}, "staff-1")
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
