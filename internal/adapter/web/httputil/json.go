package httputil

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

var (
	ErrBadJSON      = errors.New("bad json data")
	ErrExpectedJSON = errors.New("expected content-type application/json")
)

func DecodeJSON(r *http.Request, dst any) error {
	if !strings.HasPrefix(r.Header.Get("content-type"), "application/json") {
		return errors.Tracef(ErrExpectedJSON)
	}

	d := json.NewDecoder(r.Body)

	d.DisallowUnknownFields()

	if err := d.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError
		var invalidUnmarshalErr *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &invalidUnmarshalErr):
			panic(err)

		case errors.Is(err, io.EOF):
			return errors.Tracef(ErrBadJSON, "body must not be empty")

		case errors.As(err, &syntaxErr):
			return errors.Tracef(ErrBadJSON, "malformed JSON at offset %v", syntaxErr.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return errors.Tracef(ErrBadJSON, "malformed JSON")

		case errors.As(err, &unmarshalTypeErr):
			return errors.Tracef(ErrBadJSON, "invalid value for %q at offset %v", unmarshalTypeErr.Field, unmarshalTypeErr.Offset)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")

			return errors.Tracef(ErrBadJSON, "unknown field %v", fieldName)

		case errors.As(err, &maxBytesError):
			return errors.Tracef(ErrBadJSON, "request body must be no larger than %v bytes", maxBytesError.Limit)

		default:
			return errors.Tracef(ErrBadJSON, err)
		}
	}

	if err := d.Decode(&struct{}{}); err != io.EOF {
		return errors.Tracef(ErrBadJSON, "unexpected additional JSON")
	}

	return nil
}
