package logger

import "github.com/polyscone/tofu/internal/pkg/errors"

// SprintError returns a pretty printed string of an error in the
// style set in OutputStyle.
func SprintError(err error) string {
	if OutputStyle == JSON {
		return errors.SprintJSON(err)
	}

	return errors.Sprint(err)
}

// PrintError pretty prints an error in the style set in OutputStyle.
func PrintError(err error) {
	Error.Print(SprintError(err))
}
