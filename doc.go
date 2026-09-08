package mwanachamacustody

import "github.com/aosanya/mwanachama-backend-custody/models"

type (
	CustodyRepository = models.CustodyRepository
	ExportRepository  = models.ExportRepository
	ContactRepository = models.ContactRepository
	ConsentRepository = models.ConsentRepository

	ActKind              = models.ActKind
	ActClass             = models.ActClass
	StructureActLogEntry = models.StructureActLogEntry

	EventKind = models.EventKind
	EventChip = models.EventChip
	Entry     = models.Entry

	Read = models.Read

	ConsentScope = models.ConsentScope
	Language     = models.Language
	Effect       = models.Effect
	Clause       = models.Clause
	TextVersion  = models.TextVersion
)

// ErrNotFound and ErrAlreadyPublished forward to their models. sentinels of
// the same name, so a caller checking errors.Is against this package's own
// name gets the identical error value models's callers already check
// against.
var (
	ErrNotFound         = models.ErrNotFound
	ErrAlreadyPublished = models.ErrAlreadyPublished
)

const (
	ActPostWithheld        = models.ActPostWithheld
	ActPostRestored        = models.ActPostRestored
	ActReportLeftStanding  = models.ActReportLeftStanding
	ActRemovalLeftStanding = models.ActRemovalLeftStanding
	ActRoleGranted         = models.ActRoleGranted
	ActRoleRevoked         = models.ActRoleRevoked
	ActCaseEscalated       = models.ActCaseEscalated
	ActTransferredOut      = models.ActTransferredOut
	ActTransferredIn       = models.ActTransferredIn
	ActLeftStructure       = models.ActLeftStructure
	ActMembershipEnded     = models.ActMembershipEnded
	ActStructureRetired    = models.ActStructureRetired
	ActAnswerReadNamed     = models.ActAnswerReadNamed
	ActCheckStarted        = models.ActCheckStarted
)

// EventKind values named by a caller outside models/ — the gateway's own
// break-glass, role-kind and survey-question composition. Not every
// EventKind in [models.EventKinds] has an external caller; the rest stay
// reachable as models.EventXxx for this repo's own use.
const (
	EventBreakGlassWrite         = models.EventBreakGlassWrite
	EventKeysRotated             = models.EventKeysRotated
	EventIDKeyGenerated          = models.EventIDKeyGenerated
	EventRoleKindRetired         = models.EventRoleKindRetired
	EventRoleKindUnretired       = models.EventRoleKindUnretired
	EventSurveyOptionAdded       = models.EventSurveyOptionAdded
	EventSurveyQuestionVersioned = models.EventSurveyQuestionVersioned
)

// ChipOrganization and ChipPlatform are the two EventChip values a caller
// outside models/ names directly; the rest of [models.EventChips] stay
// reachable as models.ChipXxx.
const (
	ChipOrganization = models.ChipOrganization
	ChipPlatform     = models.ChipPlatform
)

// ActClasses returns every act class, in chip-row order. See
// [models.ActClasses].
func ActClasses() []ActClass { return models.ActClasses() }

// IsActClass reports whether c is a live class. See [models.IsActClass].
func IsActClass(c ActClass) bool { return models.IsActClass(c) }

// ActClassOf maps a kind to its class, or ErrUnknownActKind. See
// [models.ActClassOf].
func ActClassOf(k ActKind) (ActClass, error) { return models.ActClassOf(k) }

// ActKinds returns every kind that carries a class. See [models.ActKinds].
func ActKinds() []ActKind { return models.ActKinds() }

// EventChips returns every chip, in chip-row order. See [models.EventChips].
func EventChips() []EventChip { return models.EventChips() }

// IsEventChip reports whether c is a live chip. See [models.IsEventChip].
func IsEventChip(c EventChip) bool { return models.IsEventChip(c) }

// EventKinds returns every kind that reaches a chip. See [models.EventKinds].
func EventKinds() []EventKind { return models.EventKinds() }
