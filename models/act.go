package models

import "errors"

var ErrNotFound = errors.New("mwanachamacustody: not found")

var ErrUnknownActKind = errors.New("mwanachamacustody: act kind carries no class")

var ErrStructureRequired = errors.New("mwanachamacustody: structure act requires a structure")

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

func ActClassOf(k ActKind) (ActClass, error) {
	c, ok := actClasses[k]
	if !ok {
		return "", ErrUnknownActKind
	}
	return c, nil
}

func ActKinds() []ActKind {
	out := make([]ActKind, 0, len(actClasses))
	for k := range actClasses {
		out = append(out, k)
	}
	return out
}

type StructureActLogEntry struct {
	ID int64

	StructureID string

	OccurredAt string

	Kind  ActKind
	Class ActClass

	ActorID          string
	ActorLabel       string
	ActorStructureID string

	SubjectRef string
	SubjectID  string

	Detail map[string]any

	EscalationLevel *int
	ToStructureID   string
}
