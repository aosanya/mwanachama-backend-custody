package models

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

type ActClass string

const (
	ClassModeration   ActClass = "moderation"
	ClassEscalations  ActClass = "escalations"
	ClassRoles        ActClass = "roles"
	ClassMembership   ActClass = "membership"
	ClassStructure    ActClass = "structure"
	ClassIntelligence ActClass = "intelligence"
)

type EventKind string

const (
	EventExportStarted     EventKind = "export_started"
	EventExportInterrupted EventKind = "export_interrupted"
	EventExportResumed     EventKind = "export_resumed"
	EventExportCompleted   EventKind = "export_completed"

	EventKeysRotated          EventKind = "keys_rotated"
	EventIDKeyGenerated       EventKind = "id_key_generated"
	EventIDKeyBackupConfirmed EventKind = "id_key_backup_confirmed"
	EventPhoneSaltRotated     EventKind = "phone_salt_rotated"

	EventReplicaDisconnected EventKind = "replica_disconnected"
	EventReplicaReconnected  EventKind = "replica_reconnected"
	EventSnapshotCompleted   EventKind = "snapshot_completed"

	EventProvisioned             EventKind = "provisioned"
	EventReleaseRolledOut        EventKind = "release_rolled_out"
	EventRolloutFailed           EventKind = "rollout_failed"
	EventAgreementSigned         EventKind = "agreement_signed"
	EventHandoverCompleted       EventKind = "handover_completed"
	EventCredentialChanged       EventKind = "credential_changed"
	EventWindDownInitiated       EventKind = "wind_down_initiated"
	EventVendorAccessRevoked     EventKind = "vendor_access_revoked"
	EventBreakGlassWrite         EventKind = "break_glass_write"
	EventAgreementFieldCorrected EventKind = "agreement_field_corrected"

	EventHelpLineChanged            EventKind = "help_line_changed"
	EventOrganizationSetupCompleted EventKind = "organization_setup_completed"
	EventCapabilitiesChanged        EventKind = "capabilities_changed"
	EventRoleKindRetired            EventKind = "role_kind_retired"
	EventRoleKindUnretired          EventKind = "role_kind_unretired"
	EventDiscardTakenOver           EventKind = "discard_taken_over"
	EventSurveyOptionAdded          EventKind = "survey_option_added"
	EventSurveyQuestionVersioned    EventKind = "survey_question_versioned"

	EventThemesRecomputed EventKind = "themes_recomputed"
)

type EventChip string

const (
	ChipExports       EventChip = "exports"
	ChipKeys          EventChip = "keys"
	ChipReplicaEvents EventChip = "replica_events"
	ChipPlatform      EventChip = "platform"
	ChipOrganization  EventChip = "organization"
	ChipThemes        EventChip = "themes"
)

type Scope string

const (
	ScopeOrganization Scope = "organization"
	ScopeActor        Scope = "actor"
)

type Format string

const (
	FormatPgDump Format = "pg_dump"
	FormatCSVZip Format = "csv_zip"
)

type Status string

const (
	StatusBuilding    Status = "building"
	StatusInterrupted Status = "interrupted"
	StatusCompleted   Status = "completed"
	StatusFailed      Status = "failed"
)

type ConsentScope string

const (
	ScopeMechanics   ConsentScope = "mechanics"
	ScopeAppendix    ConsentScope = "appendix"
	ScopePublicSheet ConsentScope = "public_sheet"
)

type Language string

const (
	LanguageEnglish Language = "en"
	LanguageSwahili Language = "sw"
	LanguageLuo     Language = "luo"
)

type Effect string

const (
	EffectStrengthens Effect = "strengthens"
	EffectNeutral     Effect = "neutral"
	EffectWeakens     Effect = "weakens"
)
