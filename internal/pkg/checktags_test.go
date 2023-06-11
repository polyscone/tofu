//go:build !test

package pkg

import (
	"fmt"
	"os"
)

func init() {
	fmt.Println(`Please run tests with -tags "test"`)

	os.Exit(1)
}
