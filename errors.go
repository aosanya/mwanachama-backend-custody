package mwanachamacustody

import (
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

var ErrInvalidReference = errors.New("mwanachamacustody: invalid reference")

var ErrConflict = errors.New("mwanachamacustody: already exists")

var ErrInvalidValue = errors.New("mwanachamacustody: invalid value")

type valueError struct{ message string }

func (e *valueError) Error() string        { return e.message }
func (e *valueError) Is(target error) bool { return target == ErrInvalidValue }

func invalidValue(format string, args ...any) error {
	return &valueError{message: fmt.Sprintf(format, args...)}
}

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
			return conflict(uniqueFieldsOfPostgres(pgErr))
		default:
			return err
		}
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "UNIQUE constraint failed"):
		return conflict(uniqueFieldsOfSQLite(msg))
	case strings.Contains(msg, "FOREIGN KEY constraint failed"),
		strings.Contains(msg, "NOT NULL constraint failed"),
		strings.Contains(msg, "CHECK constraint failed"):
		return reference(sqliteConstraintOf(msg))
	default:
		return err
	}
}

func uniqueFieldsOfSQLite(msg string) string {
	list := sqliteConstraintOf(msg)
	if i := strings.LastIndex(list, " ("); i >= 0 {
		list = list[:i]
	}
	if strings.HasPrefix(list, "index ") {
		return ""
	}
	columns := strings.Split(list, ", ")
	for i, column := range columns {
		columns[i] = column[strings.LastIndex(column, ".")+1:]
	}
	return strings.Join(columns, ", ")
}

func uniqueFieldsOfPostgres(pgErr *pgconn.PgError) string {
	rest, found := strings.CutPrefix(pgErr.Detail, "Key (")
	if !found {
		return ""
	}
	columns, _, found := strings.Cut(rest, ")=")
	if !found {
		return ""
	}
	return columns
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
