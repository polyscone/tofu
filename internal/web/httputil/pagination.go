package httputil

import (
	"net/http"
	"strconv"
)

func Pagination(r *http.Request) (int, int) {
	const (
		minPage     = 1
		minSize     = 1
		maxSize     = 100
		defaultSize = 20
	)

	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = minPage
	}
	if page < minPage {
		page = minPage
	}

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = defaultSize
	}
	if size < minSize {
		size = minSize
	}
	if size > maxSize {
		size = maxSize
	}

	return page, size
}
