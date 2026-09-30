package models

type Read struct {
	ID string

	ActorID     string
	ReadBy      string
	StructureID string
	ReadAt      string

	NotifiedAt *string
}

func (r Read) Notified() bool { return r.NotifiedAt != nil }
