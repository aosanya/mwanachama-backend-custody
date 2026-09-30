package mwanachamacustody

import (
	"context"
	"errors"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

var ErrNoMechanicsInForce = errors.New("mwanachamacustody: no mechanics version in force for this language")

func Enroll(ctx context.Context, repo models.ConsentRepository, actorID, structureID string, language models.Language, now string) (models.Record, error) {
	mechanics, err := repo.GetInForce(ctx, models.ScopeMechanics, language)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.Record{}, ErrNoMechanicsInForce
		}
		return models.Record{}, err
	}

	rec := models.Record{
		ActorID:            actorID,
		MechanicsVersionID: mechanics.ID,
		AgreedAt:           now,
		Language:           language,
		StructureID:        structureID,
	}

	appendix, err := repo.GetInForce(ctx, models.ScopeAppendix, language)
	switch {
	case err == nil:
		rec.AppendixVersionID = appendix.ID
	case errors.Is(err, models.ErrNotFound):
	default:
		return models.Record{}, err
	}

	return repo.CreateRecord(ctx, rec)
}
