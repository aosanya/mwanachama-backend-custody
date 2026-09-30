package models

import "errors"

var ErrUnknownEventKind = errors.New("mwanachamacustody: event kind reaches no filter chip")

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

func EventChipOf(k EventKind) (EventChip, error) {
	c, ok := eventChips[k]
	if !ok {
		return "", ErrUnknownEventKind
	}
	return c, nil
}

func EventKinds() []EventKind {
	out := make([]EventKind, 0, len(eventChips))
	for k := range eventChips {
		out = append(out, k)
	}
	return out
}

type Entry struct {
	ID int64

	OccurredAt string

	Kind EventKind
	Chip EventChip

	ActorID    string
	ActorLabel string
	Detail     string
	Evidence   map[string]any
	SubjectID  string
}
