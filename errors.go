package mwanachamacustody

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

// ErrInvalidReference means the caller named a row that does not exist — a
// consent record's mechanics_version_id pointing at no version. Mirrors the
// gateway's internal/domain/domerr.ErrInvalidReference; this package cannot
// import that sentinel directly (it lives inside the gateway module), so it
// carries its own, same-shaped one. Callers classify with errors.Is against
// this sentinel exactly as they do against domerr's.
var ErrInvalidReference = errors.New("mwanachamacustody: invalid reference")

// ErrConflict means the row the caller asked to create already exists.
var ErrConflict = errors.New("mwanachamacustody: already exists")

// fieldError wraps a sentinel with the field at fault, so a log line can say
// which reference broke while the wire response (built from the bare
// sentinel by the caller) stays generic.
type fieldError struct {
	err   error
	field string
}

func (e *fieldError) Error() string { return e.err.Error() + ": " + e.field }
func (e *fieldError) Unwrap() error { return e.err }

// reference wraps ErrInvalidReference with the column at fault.
func reference(field string) error {
	if field == "" {
		return ErrInvalidReference
	}
	return &fieldError{err: ErrInvalidReference, field: field}
}

// conflict wraps ErrConflict with the field at fault.
func conflict(field string) error {
	if field == "" {
		return ErrConflict
	}
	return &fieldError{err: ErrConflict, field: field}
}

// SQLSTATE class 23 is "integrity constraint violation".
const (
	sqlstateNotNullViolation    = "23502"
	sqlstateForeignKeyViolation = "23503"
	sqlstateUniqueViolation     = "23505"
	sqlstateCheckViolation      = "23514"
)

// classify maps a driver error onto ErrInvalidReference/ErrConflict where it
// can, on either dialect this package runs against — mirrors
// mwanachama-backend-comm's classify byte-for-byte in shape. Postgres reports
// a structured pgconn.PgError with a SQLSTATE; sqlite (tests only) reports a
// plain-text driver error with no structured code, so that branch matches on
// the constraint-violation phrase modernc.org/sqlite's error text always
// contains. Returns err unchanged where neither matches.
func classify(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case sqlstateForeignKeyViolation, sqlstateNotNullViolation, sqlstateCheckViolation:
			return reference(constraintOf(pgErr))
		case sqlstateUniqueViolation:
			return conflict(constraintOf(pgErr))
		default:
			return err
		}
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed"):
		return conflict(sqliteConstraintOf(msg))
	case strings.Contains(msg, "FOREIGN KEY constraint failed"),
		strings.Contains(msg, "NOT NULL constraint failed"),
		strings.Contains(msg, "CHECK constraint failed"):
		return reference(sqliteConstraintOf(msg))
	default:
		return err
	}
}

// sqliteConstraintOf pulls the "table.column[, table.column...]" tail
// modernc.org/sqlite appends after "constraint failed: <KIND> constraint
// failed: " — the closest sqlite equivalent of pgconn.PgError's constraint
// name, good enough for a reference/conflict field label.
func sqliteConstraintOf(msg string) string {
	if i := strings.LastIndex(msg, "failed: "); i >= 0 {
		return msg[i+len("failed: "):]
	}
	return msg
}

func constraintOf(pgErr *pgconn.PgError) string {
	if pgErr.ConstraintName != "" {
		return pgErr.ConstraintName
	}
	if pgErr.ColumnName != "" {
		return pgErr.ColumnName
	}
	return pgErr.TableName
}
