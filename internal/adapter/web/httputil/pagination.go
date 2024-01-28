package httputil

import (
	"net/http"
	"strconv"
)

func Pagination(r *http.Request) (int, int) {
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil {
		page = 1
	}
	if page < 1 {
		page = 1
	}

	const maxSize = 100

	size, err := strconv.Atoi(r.URL.Query().Get("size"))
	if err != nil {
		size = 20
	}
	if size < 1 {
		size = 1
	}
	if size > maxSize {
		size = maxSize
	}

	return page, size
}
