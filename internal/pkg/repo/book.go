package repo

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

func NewBook[T any](page, size, totalRows int) *Book[T] {
	b := &Book[T]{
		Page: Page[T]{
			Number: page,
			Rows:   make([]T, 0, size),
		},
		PageSize:  size,
		TotalRows: totalRows,
	}

	b.update()

	return b
}

func (b *Book[T]) update() {
	rows := len(b.Page.Rows)

	if b.PageSize < rows {
		b.PageSize = rows
	}
	if b.TotalRows < rows {
		b.TotalRows = rows
	}

	b.TotalPages = b.TotalRows / b.PageSize
	if b.TotalRows%b.PageSize != 0 {
		b.TotalPages++
	}
}

func (b *Book[T]) AddRow(row T) {
	b.Page.Rows = append(b.Page.Rows, row)

	b.update()
}

func (b *Book[T]) AddRows(rows []T) {
	b.Page.Rows = append(b.Page.Rows, rows...)

	b.update()
}
