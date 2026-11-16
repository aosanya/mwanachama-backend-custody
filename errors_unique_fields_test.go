package mwanachamacustody

import (
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

func TestPostgresUniqueViolationNamesTheColumnsAndNoValuesOrIndex(t *testing.T) {
	err := classify(&pgconn.PgError{
		Code:           sqlstateUniqueViolation,
		ConstraintName: "mwanachama_0d6e4079_9386d52051ba_scope_version_language_key",
		Detail:         "Key (scope, version, language)=(mechanics, v2, en) already exists.",
	})

	if !errors.Is(err, ErrConflict) {
		t.Fatalf("err = %v, want ErrConflict", err)
	}
	if want := "mwanachamacustody: already exists: scope, version, language"; err.Error() != want {
		t.Fatalf("message = %q, want %q", err.Error(), want)
	}
}

func TestPostgresUniqueViolationWithoutDetailNamesNothingPhysical(t *testing.T) {
	err := classify(&pgconn.PgError{Code: sqlstateUniqueViolation, ConstraintName: "mwanachama_0d6e4079_9386d52051ba_pkey"})

	if want := ErrConflict.Error(); err.Error() != want {
		t.Fatalf("message = %q, want %q", err.Error(), want)
	}
}

func TestSQLiteUniqueViolationOnAnIndexNamesNothingPhysical(t *testing.T) {
	err := classify(errors.New("constraint failed: UNIQUE constraint failed: index 'by_name' (2067)"))

	if want := ErrConflict.Error(); err.Error() != want {
		t.Fatalf("message = %q, want %q", err.Error(), want)
	}
}
