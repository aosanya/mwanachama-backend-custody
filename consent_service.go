package mwanachamacustody

import (
	"context"
	"errors"

	"github.com/aosanya/mwanachama-backend-custody/models"
)

var ErrNoMechanicsInForce = errors.New("mwanachamacustody: no mechanics version in force for this language")

func Enroll(ctx context.Context, repo models.ConsentRepository, actorID, structureID string, language models.Language, now string) (models.Record, error) {
	return enrollRecord(ctx, repo, models.Record{
		ActorID:     actorID,
		AgreedAt:    now,
		Language:    language,
		StructureID: structureID,
	})
}

func enrollRecord(ctx context.Context, repo models.ConsentRepository, rec models.Record) (models.Record, error) {
	mechanics, err := repo.GetInForce(ctx, models.ScopeMechanics, rec.Language)
	if err != nil {
		if errors.Is(err, models.ErrNotFound) {
			return models.Record{}, ErrNoMechanicsInForce
		}
		return models.Record{}, err
	}
	rec.MechanicsVersionID = mechanics.ID

	appendix, err := repo.GetInForce(ctx, models.ScopeAppendix, rec.Language)
	switch {
	case err == nil:
		rec.AppendixVersionID = appendix.ID
	case errors.Is(err, models.ErrNotFound):
	default:
		return models.Record{}, err
	}

	return repo.CreateRecord(ctx, rec)
}
