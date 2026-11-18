package mwanachamacustody

import (
	"context"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

type RecordPaperConsentInput struct {
	ActorID     string          `json:"actor_id"`
	RecordedBy  string          `json:"recorded_by"`
	StructureID string          `json:"structure_id"`
	Language    models.Language `json:"language"`
}

func (m *CustodyManager) RecordPaperConsent(ctx context.Context, in RecordPaperConsentInput) (models.Record, error) {
	if in.ActorID == "" {
		return models.Record{}, invalidValue("actor_id is required")
	}
	if in.RecordedBy == "" {
		return models.Record{}, invalidValue("the one recording the consent is required")
	}
	if in.Language == "" {
		in.Language = models.LanguageEnglish
	}
	standing, found, err := m.standingAgreement(ctx, in.ActorID, in.Language)
	if err != nil {
		return models.Record{}, err
	}
	if found {
		return standing, nil
	}
	agreedAt, err := m.gate.AgreementInstant(ctx, in.ActorID)
	if err != nil {
		return models.Record{}, err
	}
	return enrollRecord(ctx, m.consent, models.Record{
		ActorID:     in.ActorID,
		StructureID: in.StructureID,
		Language:    in.Language,
		AgreedAt:    models.FormatTime(agreedAt),
		RecordedBy:  in.RecordedBy,
	})
}
