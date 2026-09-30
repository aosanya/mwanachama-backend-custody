package models

import "context"

const DefaultPage = 200

type CustodyRepository interface {
	AppendAct(ctx context.Context, e StructureActLogEntry) (StructureActLogEntry, error)

	ListActs(ctx context.Context, structures []string, class ActClass, limit int) ([]StructureActLogEntry, error)

	CountActsByClass(ctx context.Context, structures []string, since string) (map[ActClass]int, error)

	AppendEvent(ctx context.Context, e Entry) (Entry, error)

	ListEvents(ctx context.Context, chip EventChip, limit int) ([]Entry, error)

	CountEventsByChip(ctx context.Context, since string) (map[EventChip]int, error)

	GetEvent(ctx context.Context, id int64) (Entry, error)
}

type ExportRepository interface {
	Create(ctx context.Context, j Job) (Job, error)

	Get(ctx context.Context, id string) (Job, error)

	ListOrganization(ctx context.Context, limit int) ([]Job, error)

	ListForActor(ctx context.Context, actorID string, limit int) ([]Job, error)

	Progress(ctx context.Context, id string, p ProgressUpdate) (Job, error)

	Complete(ctx context.Context, id string, c Completion) (Job, error)

	MarkFileRemoved(ctx context.Context, id string, at string) (Job, error)

	DueForRemoval(ctx context.Context, asOf string, limit int) ([]Job, error)
}

type ContactRepository interface {
	Create(ctx context.Context, r Read) (Read, error)

	Get(ctx context.Context, id string) (Read, error)

	ListForSubject(ctx context.Context, actorID string) ([]Read, error)

	ListByOperator(ctx context.Context, operatorID string) ([]Read, error)
}

type ConsentRepository interface {
	CreateVersion(ctx context.Context, v TextVersion) (TextVersion, error)

	PublishVersion(ctx context.Context, id, publishedBy string, now string) (TextVersion, error)

	GetVersion(ctx context.Context, id string) (TextVersion, error)

	GetInForce(ctx context.Context, scope ConsentScope, language Language) (TextVersion, error)

	ListVersions(ctx context.Context, scope ConsentScope, language Language) ([]TextVersion, error)

	CreateRecord(ctx context.Context, r Record) (Record, error)

	GetRecord(ctx context.Context, id string) (Record, error)

	ListForActor(ctx context.Context, actorID string) ([]Record, error)

	CurrentForActor(ctx context.Context, actorID string) (Record, error)
}
