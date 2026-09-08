package models

import (
	"errors"
	"time"
)

// ErrUnknownEventKind is the organization-scoped sibling of
// ErrUnknownActKind — a kind with no chip cannot be appended.
var ErrUnknownEventKind = errors.New("mwanachamacustody: event kind reaches no filter chip")

// EventKind is what was done to the organization's data — mirrors
// mwanachama-backend-api-gateway's internal/domain/custody.EventKind
// field-for-field. Deliberately stated without a count in prose anywhere:
// the census in the tests is the figure of record.
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

	EventProvisioned       EventKind = "provisioned"
	EventReleaseRolledOut  EventKind = "release_rolled_out"
	EventRolloutFailed     EventKind = "rollout_failed"
	EventAgreementSigned   EventKind = "agreement_signed"
	EventHandoverCompleted EventKind = "handover_completed"
	EventCredentialChanged EventKind = "credential_changed"

	EventWindDownInitiated   EventKind = "wind_down_initiated"
	EventVendorAccessRevoked EventKind = "vendor_access_revoked"

	EventBreakGlassWrite EventKind = "break_glass_write"

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

// EventChip is the filter grouping on M85's chip row.
type EventChip string

const (
	ChipExports       EventChip = "exports"
	ChipKeys          EventChip = "keys"
	ChipReplicaEvents EventChip = "replica_events"
	ChipPlatform      EventChip = "platform"
	ChipOrganization  EventChip = "organization"
	ChipThemes        EventChip = "themes"
)

// eventChips is the fixed map, and every EventKind above appears exactly
// once.
var eventChips = map[EventKind]EventChip{
	EventExportStarted:     ChipExports,
	EventExportInterrupted: ChipExports,
	EventExportResumed:     ChipExports,
	EventExportCompleted:   ChipExports,

	EventKeysRotated:          ChipKeys,
	EventIDKeyGenerated:       ChipKeys,
	EventIDKeyBackupConfirmed: ChipKeys,
	EventPhoneSaltRotated:     ChipKeys,

	EventReplicaDisconnected: ChipReplicaEvents,
	EventReplicaReconnected:  ChipReplicaEvents,
	EventSnapshotCompleted:   ChipReplicaEvents,

	EventProvisioned:             ChipPlatform,
	EventReleaseRolledOut:        ChipPlatform,
	EventRolloutFailed:           ChipPlatform,
	EventAgreementSigned:         ChipPlatform,
	EventHandoverCompleted:       ChipPlatform,
	EventCredentialChanged:       ChipPlatform,
	EventWindDownInitiated:       ChipPlatform,
	EventVendorAccessRevoked:     ChipPlatform,
	EventBreakGlassWrite:         ChipPlatform,
	EventAgreementFieldCorrected: ChipPlatform,

	EventHelpLineChanged:            ChipOrganization,
	EventOrganizationSetupCompleted: ChipOrganization,
	EventCapabilitiesChanged:        ChipOrganization,
	EventRoleKindRetired:            ChipOrganization,
	EventRoleKindUnretired:          ChipOrganization,
	EventDiscardTakenOver:           ChipOrganization,
	EventSurveyOptionAdded:          ChipOrganization,
	EventSurveyQuestionVersioned:    ChipOrganization,

	EventThemesRecomputed: ChipThemes,
}

// allEventChips is M85's chip order — `Exports · Keys · Replica events ·
// Platform · Organization · Themes`.
var allEventChips = []EventChip{
	ChipExports, ChipKeys, ChipReplicaEvents,
	ChipPlatform, ChipOrganization, ChipThemes,
}

// EventChips returns every chip, in the order M85 draws them.
func EventChips() []EventChip { return append([]EventChip(nil), allEventChips...) }

// IsEventChip reports whether c is a live chip.
func IsEventChip(c EventChip) bool {
	for _, known := range allEventChips {
		if known == c {
			return true
		}
	}
	return false
}

// EventChipOf maps a kind to its chip, or returns ErrUnknownEventKind.
func EventChipOf(k EventKind) (EventChip, error) {
	c, ok := eventChips[k]
	if !ok {
		return "", ErrUnknownEventKind
	}
	return c, nil
}

// EventKinds returns every kind that reaches a chip.
func EventKinds() []EventKind {
	out := make([]EventKind, 0, len(eventChips))
	for k := range eventChips {
		out = append(out, k)
	}
	return out
}

// Entry is one row of the organization's custody log.
type Entry struct {
	// ID is monotonic; read newest-first, never re-ordered.
	ID int64

	OccurredAt time.Time

	Kind EventKind
	Chip EventChip

	// ActorID empty means the system, or a vendor operator — who has no row
	// in this database at all.
	ActorID string
	// ActorLabel is required: the name as it read at the time.
	ActorLabel string
	Detail     string
	// Evidence is the checkable artifact: a checksum, a key-id transition, a
	// duration, a diff. Nil where a row has none, drawn as `—`.
	Evidence map[string]any
	// SubjectID is what the row is *about*, loosely.
	SubjectID string
}
