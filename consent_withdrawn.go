package mwanachamacustody

import (
	"context"
	"errors"
)

func (m *CustodyManager) WithdrawnConsents(ctx context.Context) ([]Withdrawal, error) {
	actors, err := m.ledger.Actors(ctx)
	if err != nil {
		return nil, err
	}
	var withdrawn []Withdrawal
	for _, actor := range actors {
		checked := m.gate.Check(ctx, actor)
		if errors.Is(checked, ErrConsentWithdrawn) {
			latest, _, err := m.ledger.Latest(ctx, actor)
			if err != nil {
				return nil, err
			}
			withdrawn = append(withdrawn, latest)
			continue
		}
		if checked != nil && !errors.Is(checked, ErrConsentRequired) {
			return nil, checked
		}
	}
	return withdrawn, nil
}
