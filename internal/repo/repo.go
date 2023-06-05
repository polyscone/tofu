package repo

import (
	"github.com/polyscone/tofu/internal/pkg/errors"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrConflict     = errors.New("conflict")
	ErrInvalidInput = errors.New("invalid input")
)

type ConflictError struct {
	errors.Map
}

func (c ConflictError) Error() string {
	return c.Map.String()
}

type Page[T any] struct {
	Number int
	Rows   []T
}

type Book[T any] struct {
	Page       Page[T]
	PageSize   int
	TotalRows  int
	TotalPages int
}

func NewBook[T any](rows []T, page, size, totalRows int) *Book[T] {
	b := &Book[T]{
		Page: Page[T]{
			Number: page,
			Rows:   rows,
		},
		PageSize:  size,
		TotalRows: totalRows,
	}

	nRows := len(b.Page.Rows)
	if b.PageSize < nRows {
		b.PageSize = nRows
	}
	if b.TotalRows < nRows {
		b.TotalRows = nRows
	}

	b.TotalPages = b.TotalRows / b.PageSize
	if b.TotalRows%b.PageSize != 0 {
		b.TotalPages++
	}

	return b
}
