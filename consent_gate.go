package mwanachamacustody

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/aosanya/mwanachama-backend-custody/models"
	"github.com/aosanya/mwanachama-backend-shared/consent"
)

var ErrConsentRequired = consent.ErrRequired

var ErrConsentWithdrawn = consent.ErrWithdrawn

type ConsentGate interface {
	Check(ctx context.Context, actorID string) error
}

type Withdrawal struct {
	ActorID     string `json:"actor_id"`
	WithdrawnAt string `json:"withdrawn_at"`
}

type WithdrawalSubscriber func(ctx context.Context, w Withdrawal) error

type WithdrawalNotifier interface {
	Subscribe(fn WithdrawalSubscriber)
}

type WithdrawalLedger interface {
	Record(ctx context.Context, w Withdrawal) error
	Latest(ctx context.Context, actorID string) (Withdrawal, bool, error)
}

type MemoryWithdrawalLedger struct {
	mu   sync.Mutex
	rows map[string]Withdrawal
}

func NewMemoryWithdrawalLedger() *MemoryWithdrawalLedger {
	return &MemoryWithdrawalLedger{rows: map[string]Withdrawal{}}
}

func (l *MemoryWithdrawalLedger) Record(_ context.Context, w Withdrawal) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.rows[w.ActorID] = w
	return nil
}

func (l *MemoryWithdrawalLedger) Latest(_ context.Context, actorID string) (Withdrawal, bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	w, ok := l.rows[actorID]
	return w, ok, nil
}

type Consent struct {
	repo   models.ConsentRepository
	ledger WithdrawalLedger
	now    Clock

	mu   sync.Mutex
	subs []WithdrawalSubscriber
}

func NewConsent(repo models.ConsentRepository, ledger WithdrawalLedger, now Clock) *Consent {
	if now == nil {
		now = SystemClock
	}
	return &Consent{repo: repo, ledger: ledger, now: now}
}

func (c *Consent) Subscribe(fn WithdrawalSubscriber) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.subs = append(c.subs, fn)
}

func (c *Consent) Check(ctx context.Context, actorID string) error {
	rec, err := c.repo.CurrentForActor(ctx, actorID)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return ErrConsentRequired
		}
		return err
	}
	w, found, err := c.ledger.Latest(ctx, actorID)
	if err != nil {
		return err
	}
	if !found {
		return nil
	}
	agreed, err := models.ParseTime(rec.AgreedAt)
	if err != nil {
		return err
	}
	withdrawn, err := models.ParseTime(w.WithdrawnAt)
	if err != nil {
		return err
	}
	if agreed.After(withdrawn) {
		return nil
	}
	return ErrConsentWithdrawn
}

func (c *Consent) AgreementInstant(ctx context.Context, actorID string) (time.Time, error) {
	now := c.now()
	w, found, err := c.ledger.Latest(ctx, actorID)
	if err != nil || !found {
		return now, err
	}
	withdrawn, err := models.ParseTime(w.WithdrawnAt)
	if err != nil {
		return time.Time{}, err
	}
	if now.After(withdrawn) {
		return now, nil
	}
	return withdrawn.Add(time.Nanosecond), nil
}

func (c *Consent) Withdraw(ctx context.Context, actorID string) (Withdrawal, error) {
	if actorID == "" {
		return Withdrawal{}, ErrConsentRequired
	}
	w := Withdrawal{ActorID: actorID, WithdrawnAt: models.FormatTime(c.now())}
	if err := c.ledger.Record(ctx, w); err != nil {
		return Withdrawal{}, err
	}
	c.mu.Lock()
	subs := append([]WithdrawalSubscriber(nil), c.subs...)
	c.mu.Unlock()
	var failed []error
	for _, fn := range subs {
		if err := fn(ctx, w); err != nil {
			failed = append(failed, err)
		}
	}
	return w, errors.Join(failed...)
}

func Gate(ctx context.Context, g ConsentGate, actorID string) error {
	if g == nil {
		return nil
	}
	return g.Check(ctx, actorID)
}

var (
	_ ConsentGate        = (*Consent)(nil)
	_ WithdrawalNotifier = (*Consent)(nil)
)
