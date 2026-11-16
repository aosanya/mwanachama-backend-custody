package mwanachamacustody

import (
	"context"
	"errors"
)

type ConsentStatus string

const (
	ConsentStatusNone      ConsentStatus = "none"
	ConsentStatusAgreed    ConsentStatus = "agreed"
	ConsentStatusWithdrawn ConsentStatus = "withdrawn"
)

type ConsentState struct {
	Status             ConsentStatus `json:"status"`
	AgreedAt           string        `json:"agreed_at,omitempty"`
	MechanicsVersionID string        `json:"mechanics_version_id,omitempty"`
	Language           string        `json:"language,omitempty"`
}

func (m *CustodyManager) GetMyConsent(ctx context.Context, actorID string) (ConsentState, error) {
	if actorID == "" {
		return ConsentState{}, ErrConsentRequired
	}
	checked := m.gate.Check(ctx, actorID)
	if errors.Is(checked, ErrConsentRequired) {
		return ConsentState{Status: ConsentStatusNone}, nil
	}
	if checked != nil && !errors.Is(checked, ErrConsentWithdrawn) {
		return ConsentState{}, checked
	}
	rec, err := m.consent.CurrentForActor(ctx, actorID)
	if err != nil {
		return ConsentState{}, err
	}
	status := ConsentStatusAgreed
	if checked != nil {
		status = ConsentStatusWithdrawn
	}
	return ConsentState{Status: status, AgreedAt: rec.AgreedAt, MechanicsVersionID: rec.MechanicsVersionID, Language: string(rec.Language)}, nil
}
