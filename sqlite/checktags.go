//go:build !fts5 || !json1

package sqlite

import (
	"fmt"
	"os"
)

func init() {
	fmt.Println(`Please rebuild with -tags "fts5 json1"`)

	os.Exit(1)
}
