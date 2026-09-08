// Package models holds the domain types for every cluster this repo houses:
// act.go and event.go (custody), export.go/progress.go (export), contact.go
// (contact), consent.go (consent). Ported field-for-field from
// mwanachama-backend-api-gateway's internal/domain/{custody,export,contact,
// consent} packages — see this repo's CLAUDE.md for what changed along the
// way.
package models

import (
	"errors"
	"time"
)

// ErrNotFound is returned when an entry id, job id, read id or version/record
// id names no row. One sentinel shared across the whole package rather than
// one per domain — every caller here already type-switches on which method it
// called, not on which error came back.
var ErrNotFound = errors.New("mwanachamacustody: not found")

// ErrUnknownActKind is returned by ActClassOf for a kind carrying no class
// arm — G190 rendered in Go. A kind added without an arm refuses at the
// write, before a row exists to be miscounted, rather than landing with a
// silently null class. See ActClassOf.
var ErrUnknownActKind = errors.New("mwanachamacustody: act kind carries no class")

var ErrStructureRequired = errors.New("mwanachamacustody: structure act requires a structure")

type ActKind string

const (
	ActPostWithheld        ActKind = "post_withheld"
	ActPostRestored        ActKind = "post_restored"
	ActReportLeftStanding  ActKind = "report_left_standing"
	ActRemovalLeftStanding ActKind = "removal_left_standing"
	ActRoleGranted         ActKind = "role_granted"
	ActRoleRevoked         ActKind = "role_revoked"
	ActCaseEscalated       ActKind = "case_escalated"
	ActTransferredOut      ActKind = "transferred_out"
	ActTransferredIn       ActKind = "transferred_in"
	ActLeftStructure       ActKind = "left_structure"
	ActMembershipEnded     ActKind = "membership_ended"
	ActStructureRetired    ActKind = "structure_retired"
	ActAnswerReadNamed     ActKind = "answer_read_named"
	ActCheckStarted        ActKind = "check_started"
)

// ActClass is the filter grouping on M106's chip row. Stored rather than
// computed at read time so the filter is an index scan.
type ActClass string

const (
	ClassModeration   ActClass = "moderation"
	ClassRoles        ActClass = "roles"
	ClassEscalations  ActClass = "escalations"
	ClassMembership   ActClass = "membership"
	ClassStructure    ActClass = "structure"
	ClassIntelligence ActClass = "intelligence"
)

// actClasses is the fixed map. Every ActKind above appears exactly once —
// see ActClassOf.
var actClasses = map[ActKind]ActClass{
	ActPostWithheld:        ClassModeration,
	ActPostRestored:        ClassModeration,
	ActReportLeftStanding:  ClassModeration,
	ActRemovalLeftStanding: ClassModeration,

	ActRoleGranted: ClassRoles,
	ActRoleRevoked: ClassRoles,

	ActCaseEscalated: ClassEscalations,

	ActTransferredOut:  ClassMembership,
	ActTransferredIn:   ClassMembership,
	ActLeftStructure:   ClassMembership,
	ActMembershipEnded: ClassMembership,
	ActCheckStarted:    ClassMembership,

	ActStructureRetired: ClassStructure,

	ActAnswerReadNamed: ClassIntelligence,
}

var allActClasses = []ActClass{
	ClassModeration, ClassEscalations, ClassRoles,
	ClassMembership, ClassStructure, ClassIntelligence,
}

// ActClasses returns every class, in the order M106 draws its chips.
func ActClasses() []ActClass { return append([]ActClass(nil), allActClasses...) }

// IsActClass reports whether c is a live class.
func IsActClass(c ActClass) bool {
	for _, known := range allActClasses {
		if known == c {
			return true
		}
	}
	return false
}

// ActClassOf maps a kind to its class, or returns ErrUnknownActKind.
func ActClassOf(k ActKind) (ActClass, error) {
	c, ok := actClasses[k]
	if !ok {
		return "", ErrUnknownActKind
	}
	return c, nil
}

// ActKinds returns every kind that carries a class, for tests and for the
// census that proves the map is total.
func ActKinds() []ActKind {
	out := make([]ActKind, 0, len(actClasses))
	for k := range actClasses {
		out = append(out, k)
	}
	return out
}

type StructureActLogEntry struct {
	// ID is monotonic. The log is read newest-first and never re-ordered.
	ID int64

	StructureID string

	OccurredAt time.Time

	Kind  ActKind
	Class ActClass

	// ActorID empty means the timer — a real state, not a missing value.
	ActorID string
	// ActorLabel is the name and role class as they read at the time.
	ActorLabel       string
	ActorStructureID string

	// SubjectRef is what the row is about as the reader sees it — text, not a
	// foreign key, so it keeps rendering after the object is retired,
	// re-parented or revoked.
	SubjectRef string
	// SubjectID is the object's id where one exists. Loose: the log outlives
	// what it describes.
	SubjectID string

	// Detail is the human sentence, composed by the writer. Nullable — two
	// years after OccurredAt the free-text keys are stripped and the column
	// collapses to nil when nothing survives; the structured keys are
	// permanent.
	Detail map[string]any

	// EscalationLevel is set on ActCaseEscalated only.
	EscalationLevel *int
	ToStructureID   string
}
