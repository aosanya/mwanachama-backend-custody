package mwanachamacustody

import (
	"context"
	"fmt"
	"regexp"

	"gorm.io/gorm"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

var ledgerTableName = regexp.MustCompile(`^[a-z0-9_]+$`)

type DBWithdrawalLedger struct {
	db    *gorm.DB
	table string
}

func NewDBWithdrawalLedger(db *gorm.DB, table string) (*DBWithdrawalLedger, error) {
	if db == nil {
		return nil, fmt.Errorf("NewDBWithdrawalLedger: db must not be nil")
	}
	if !ledgerTableName.MatchString(table) {
		return nil, fmt.Errorf("NewDBWithdrawalLedger: table %q is not a plain name", table)
	}
	ddl := "CREATE TABLE IF NOT EXISTS " + table + " (actor_id TEXT NOT NULL, withdrawn_at TEXT NOT NULL)"
	if err := db.Exec(ddl).Error; err != nil {
		return nil, fmt.Errorf("NewDBWithdrawalLedger: %w", err)
	}
	idx := "CREATE INDEX IF NOT EXISTS " + table + "_actor_idx ON " + table + " (actor_id)"
	if err := db.Exec(idx).Error; err != nil {
		return nil, fmt.Errorf("NewDBWithdrawalLedger: %w", err)
	}
	return &DBWithdrawalLedger{db: db, table: table}, nil
}

func (l *DBWithdrawalLedger) Record(ctx context.Context, w Withdrawal) error {
	return l.db.WithContext(ctx).Table(l.table).Create(map[string]any{
		"actor_id":     w.ActorID,
		"withdrawn_at": w.WithdrawnAt,
	}).Error
}

func (l *DBWithdrawalLedger) Latest(ctx context.Context, actorID string) (Withdrawal, bool, error) {
	var stamps []string
	if err := l.db.WithContext(ctx).Table(l.table).Where("actor_id = ?", actorID).Pluck("withdrawn_at", &stamps).Error; err != nil {
		return Withdrawal{}, false, err
	}
	best := ""
	for _, s := range stamps {
		if best == "" {
			best = s
			continue
		}
		a, err := models.ParseTime(s)
		if err != nil {
			return Withdrawal{}, false, err
		}
		b, err := models.ParseTime(best)
		if err != nil {
			return Withdrawal{}, false, err
		}
		if a.After(b) {
			best = s
		}
	}
	if best == "" {
		return Withdrawal{}, false, nil
	}
	return Withdrawal{ActorID: actorID, WithdrawnAt: best}, true, nil
}

var _ WithdrawalLedger = (*DBWithdrawalLedger)(nil)
