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

	Scope          = models.Scope
	Format         = models.Format
	Status         = models.Status
	Job            = models.Job
	ProgressUpdate = models.ProgressUpdate
	Completion     = models.Completion

	ConsentScope = models.ConsentScope
	Language     = models.Language
	Effect       = models.Effect
	Clause       = models.Clause
	TextVersion  = models.TextVersion
	Record       = models.Record
)

var (
	ErrNotFound              = models.ErrNotFound
	ErrAlreadyPublished      = models.ErrAlreadyPublished
	ErrInvalid               = models.ErrInvalid
	ErrUnknownActKind        = models.ErrUnknownActKind
	ErrUnknownEventKind      = models.ErrUnknownEventKind
	ErrStructureRequired     = models.ErrStructureRequired
	ErrResumeAlreadyRecorded = models.ErrResumeAlreadyRecorded
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

const (
	EventBreakGlassWrite         = models.EventBreakGlassWrite
	EventKeysRotated             = models.EventKeysRotated
	EventIDKeyGenerated          = models.EventIDKeyGenerated
	EventRoleKindRetired         = models.EventRoleKindRetired
	EventRoleKindUnretired       = models.EventRoleKindUnretired
	EventSurveyOptionAdded       = models.EventSurveyOptionAdded
	EventSurveyQuestionVersioned = models.EventSurveyQuestionVersioned
)

const (
	ChipOrganization = models.ChipOrganization
	ChipPlatform     = models.ChipPlatform
)

func ActClassOf(k ActKind) (ActClass, error) { return models.ActClassOf(k) }

func ActKinds() []ActKind { return models.ActKinds() }

func EventKinds() []EventKind { return models.EventKinds() }
