package models

import "errors"

var ErrAlreadyPublished = errors.New("mwanachamacustody: consent version already published")

type Clause struct {
	Key  string `json:"key"`
	Text string `json:"text"`
}

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

	PublishedAt  *string `json:"published_at,omitempty"`
	PublishedBy  string  `json:"published_by,omitempty"`
	SupersededAt *string `json:"superseded_at,omitempty"`
}

type Record struct {
	ID                 string   `json:"id"`
	ActorID            string   `json:"actor_id"`
	MechanicsVersionID string   `json:"mechanics_version_id"`
	AppendixVersionID  string   `json:"appendix_version_id,omitempty"`
	AgreedAt           string   `json:"agreed_at"`
	Language           Language `json:"language"`
	StructureID        string   `json:"structure_id"`
}
