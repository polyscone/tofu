package sqlite

import (
	"database/sql"
	"fmt"
	"time"
)

type NullableTime interface {
	IsZero() bool
	UTC() time.Time
}

func NewNullTime(t NullableTime) sql.NullTime {
	var nullable sql.NullTime
	if t.IsZero() {
		return nullable
	}

	switch t := t.(type) {
	case time.Time:
		nullable.Time = t

	case *time.Time:
		nullable.Time = *t

	default:
		panic(fmt.Sprintf("expected either time.Time or *time.Time; got %T", t))
	}

	nullable.Valid = true

	return nullable
}
