package repository

import (
	"errors"

	"github.com/polyscone/tofu/internal/pkg/errsx"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("conflict")
	ErrLogin    = errors.New("login")
)

type InputError string

func (i InputError) Error() string {
	return string(i)
}

type ConflictError struct {
	errsx.Map
}

func (c ConflictError) Error() string {
	return c.Map.String()
}

func (c ConflictError) Unwrap() error {
	return c.Map
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
	if b.TotalRows%b.PageSize > 0 {
		b.TotalPages++
	}

	return b
}
