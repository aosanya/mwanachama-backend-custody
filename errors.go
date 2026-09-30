package mwanachamacustody

import (
	"errors"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrInvalidReference = errors.New("mwanachamacustody: invalid reference")

var ErrConflict = errors.New("mwanachamacustody: already exists")

type fieldError struct {
	err   error
	field string
}

func (e *fieldError) Error() string { return e.err.Error() + ": " + e.field }
func (e *fieldError) Unwrap() error { return e.err }

func reference(field string) error {
	if field == "" {
		return ErrInvalidReference
	}
	return &fieldError{err: ErrInvalidReference, field: field}
}

func conflict(field string) error {
	if field == "" {
		return ErrConflict
	}
	return &fieldError{err: ErrConflict, field: field}
}

var ErrUnknownScope = errors.New(`mwanachamacustody: scope must be "structure" or "subtree"`)

var ErrUnknownActClass = errors.New("mwanachamacustody: unknown act class")

var ErrUnknownEventChip = errors.New("mwanachamacustody: unknown custody chip")

var ErrInvalidLimit = errors.New("mwanachamacustody: limit must be a positive integer")

var ErrInvalidSince = errors.New("mwanachamacustody: since must be an RFC3339 timestamp")

var ErrStructureIDRequired = errors.New("mwanachamacustody: structure id required")

var ErrNotSelf = errors.New("mwanachamacustody: only the actor this record is about may read it")

const (
	sqlstateNotNullViolation    = "23502"
	sqlstateForeignKeyViolation = "23503"
	sqlstateUniqueViolation     = "23505"
	sqlstateCheckViolation      = "23514"
)

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
