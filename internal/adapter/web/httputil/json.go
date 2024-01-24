package httputil

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
)

var (
	ErrBadJSON      = errors.New("bad json data")
	ErrExpectedJSON = errors.New("expected content-type application/json")
)

func decodeJSON(dst any, r io.Reader, disallowUnknownFields bool) error {
	d := json.NewDecoder(r)

	if disallowUnknownFields {
		d.DisallowUnknownFields()
	}

	if err := d.Decode(dst); err != nil {
		var syntaxErr *json.SyntaxError
		var unmarshalTypeErr *json.UnmarshalTypeError
		var invalidUnmarshalErr *json.InvalidUnmarshalError
		var maxBytesError *http.MaxBytesError

		switch {
		case errors.As(err, &invalidUnmarshalErr):
			panic(err)

		case errors.Is(err, io.EOF):
			return fmt.Errorf("%w: body must not be empty", ErrBadJSON)

		case errors.As(err, &syntaxErr):
			return fmt.Errorf("%w: malformed JSON at offset %v", ErrBadJSON, syntaxErr.Offset)

		case errors.Is(err, io.ErrUnexpectedEOF):
			return fmt.Errorf("%w: malformed JSON", ErrBadJSON)

		case errors.As(err, &unmarshalTypeErr):
			return fmt.Errorf("%w: invalid value for %q at offset %v", ErrBadJSON, unmarshalTypeErr.Field, unmarshalTypeErr.Offset)

		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")

			return fmt.Errorf("%w: unknown field %v", ErrBadJSON, fieldName)

		case errors.As(err, &maxBytesError):
			return fmt.Errorf("%w: request body must be no larger than %v bytes", ErrBadJSON, maxBytesError.Limit)

		default:
			return fmt.Errorf("%w: %w", ErrBadJSON, err)
		}
	}

	if err := d.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("%w: unexpected additional JSON", ErrBadJSON)
	}

	return nil
}

func DecodeJSON(dst any, r io.Reader) error {
	const disallowUnknownFields = true

	return decodeJSON(dst, r, disallowUnknownFields)
}

func RelaxedDecodeJSON(dst any, r io.Reader) error {
	const disallowUnknownFields = false

	return decodeJSON(dst, r, disallowUnknownFields)
}

func DecodeRequestJSON(dst any, r *http.Request) error {
	if !strings.HasPrefix(r.Header.Get("content-type"), "application/json") {
		return ErrExpectedJSON
	}

	const disallowUnknownFields = true

	return decodeJSON(dst, r.Body, disallowUnknownFields)
}

func RelaxedDecodeRequestJSON(dst any, r *http.Request) error {
	if !strings.HasPrefix(r.Header.Get("content-type"), "application/json") {
		return ErrExpectedJSON
	}

	const disallowUnknownFields = false

	return decodeJSON(dst, r.Body, disallowUnknownFields)
}
