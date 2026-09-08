package gormstore

import (
	"encoding/json"
	"time"

	"github.com/aosanya/mwanachama-backend-custody/models"
	"gorm.io/gorm"
)

// JobRow is the GORM row for a [models.Job].
//
// Datasets and RowCounts are stored as plain JSON text rather than a native
// array/jsonb column with a typed wrapper: this repo's own gormstore.Migrate
// runs AutoMigrate against both Postgres and sqlite (tests), and a portable
// column shape that both dialects render the same way is worth more here
// than a Postgres-only text[] would be.
type JobRow struct {
	ID string `gorm:"primaryKey"`

	Scope           string `gorm:"index"`
	SubjectMemberID string `gorm:"index"`
	RequestedBy     string

	RequestedAt time.Time
	SnapshotAt  time.Time

	IncludeIDs bool
	Format     string
	Datasets   string

	Status           string
	ProgressPct      *int
	ResumedFromPct   *int
	RowCounts        string
	Bytes            *int64
	Checksum         string
	StoragePath      string
	FileRemovedAt    *time.Time
	FileRemovedDueAt time.Time
}

// BeforeCreate mints an id via mintID when the caller left one unset.
func (r *JobRow) BeforeCreate(tx *gorm.DB) error {
	if r.ID == "" {
		id, err := mintID(tx, "exportjob", "custody_export_job_seq")
		if err != nil {
			return err
		}
		r.ID = id
	}
	return nil
}

// JobToRow converts a domain Job to its row shape.
func JobToRow(j models.Job) (JobRow, error) {
	datasets, err := marshalStrings(j.Datasets)
	if err != nil {
		return JobRow{}, err
	}
	counts, err := MarshalCounts(j.RowCounts)
	if err != nil {
		return JobRow{}, err
	}
	return JobRow{
		ID:               j.ID,
		Scope:            string(j.Scope),
		SubjectMemberID:  j.SubjectActorID,
		RequestedBy:      j.RequestedBy,
		RequestedAt:      j.RequestedAt,
		SnapshotAt:       j.SnapshotAt,
		IncludeIDs:       j.IncludeIDs,
		Format:           string(j.Format),
		Datasets:         datasets,
		Status:           string(j.Status),
		ProgressPct:      j.ProgressPct,
		ResumedFromPct:   j.ResumedFromPct,
		RowCounts:        counts,
		Bytes:            j.Bytes,
		Checksum:         j.Checksum,
		StoragePath:      j.StoragePath,
		FileRemovedAt:    j.FileRemovedAt,
		FileRemovedDueAt: j.FileRemovedDueAt,
	}, nil
}

// JobFromRow converts a row back to the domain Job.
func JobFromRow(r JobRow) (models.Job, error) {
	datasets, err := unmarshalStrings(r.Datasets)
	if err != nil {
		return models.Job{}, err
	}
	counts, err := UnmarshalCounts(r.RowCounts)
	if err != nil {
		return models.Job{}, err
	}
	return models.Job{
		ID:               r.ID,
		Scope:            models.Scope(r.Scope),
		SubjectActorID:   r.SubjectMemberID,
		RequestedBy:      r.RequestedBy,
		RequestedAt:      r.RequestedAt,
		SnapshotAt:       r.SnapshotAt,
		IncludeIDs:       r.IncludeIDs,
		Format:           models.Format(r.Format),
		Datasets:         datasets,
		Status:           models.Status(r.Status),
		ProgressPct:      r.ProgressPct,
		ResumedFromPct:   r.ResumedFromPct,
		RowCounts:        counts,
		Bytes:            r.Bytes,
		Checksum:         r.Checksum,
		StoragePath:      r.StoragePath,
		FileRemovedAt:    r.FileRemovedAt,
		FileRemovedDueAt: r.FileRemovedDueAt,
	}, nil
}

func marshalStrings(in []string) (string, error) {
	if len(in) == 0 {
		return "", nil
	}
	b, err := json.Marshal(in)
	return string(b), err
}

func unmarshalStrings(in string) ([]string, error) {
	if in == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(in), &out); err != nil {
		return nil, err
	}
	return out, nil
}

func MarshalCounts(in map[string]int) (string, error) {
	if len(in) == 0 {
		return "", nil
	}
	b, err := json.Marshal(in)
	return string(b), err
}

func UnmarshalCounts(in string) (map[string]int, error) {
	if in == "" {
		return nil, nil
	}
	var out map[string]int
	if err := json.Unmarshal([]byte(in), &out); err != nil {
		return nil, err
	}
	return out, nil
}
