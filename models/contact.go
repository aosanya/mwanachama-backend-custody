package models

import "time"

type Read struct {
	// ID is the event's identity; one per read.
	ID string

	ActorID string
	// ReadBy is the operator who spent the capability.
	ReadBy      string
	StructureID string
	// ReadAt is when, to the minute.
	ReadAt time.Time

	NotifiedAt *time.Time
}

func (r Read) Notified() bool { return r.NotifiedAt != nil }
