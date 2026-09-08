package models

import (
	"errors"
	"time"
)

// ErrAlreadyPublished is returned when a version already carries a
// PublishedAt. A version is a fact once published and is never edited in
// place.
var ErrAlreadyPublished = errors.New("mwanachamacustody: consent version already published")

// ConsentScope is which half of the sheet a version belongs to. Named
// distinctly from export's Scope/contact's package-level names to avoid a
// collision within this one models package (the source packages were
// separate; this one is not).
type ConsentScope string

const (
	ScopeMechanics   ConsentScope = "mechanics"
	ScopeAppendix    ConsentScope = "appendix"
	ScopePublicSheet ConsentScope = "public_sheet"
)

// Language is the rendering a version or record is in.
type Language string

const (
	LanguageEnglish Language = "en"
	LanguageSwahili Language = "sw"
	LanguageLuo     Language = "luo"
)

// Effect classifies one clause's change against the immediately prior
// version in the same language.
type Effect string

const (
	EffectStrengthens Effect = "strengthens"
	EffectNeutral     Effect = "neutral"
	EffectWeakens     Effect = "weakens"
)

// Clause is one entry of a version's clause list. Key is stable across
// versions of the same scope, so a reword reads as a change to that clause
// rather than a delete plus an insert.
type Clause struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

// TextVersion is one (Scope, Version, Language) row of the consent sheet.
// Mirrors mwanachama-backend-api-gateway's internal/domain/consent.TextVersion
// field-for-field.
type TextVersion struct {
	ID       string       `json:"id"`
	Scope    ConsentScope `json:"scope"`
	Version  string       `json:"version"`
	Language Language     `json:"language"`
	Clauses  []Clause     `json:"clauses"`

	Effect            map[string]Effect `json:"effect,omitempty"`
	Translated        bool              `json:"translated"`
	ReleaseRef        string            `json:"release_ref,omitempty"`
	CopiedMechanicsID string            `json:"copied_mechanics_id,omitempty"`

	PublishedAt  *time.Time `json:"published_at,omitempty"`
	PublishedBy  string     `json:"published_by,omitempty"`
	SupersededAt *time.Time `json:"superseded_at,omitempty"`
}

type Record struct {
	ID                 string `json:"id"`
	ActorID            string `json:"actor_id"`
	MechanicsVersionID string `json:"mechanics_version_id"`
	// AppendixVersionID is nullable: an organization that has published no
	// appendix has a one-half sheet.
	AppendixVersionID string    `json:"appendix_version_id,omitempty"`
	AgreedAt          time.Time `json:"agreed_at"`
	Language          Language  `json:"language"`
	StructureID       string    `json:"structure_id"`
}
