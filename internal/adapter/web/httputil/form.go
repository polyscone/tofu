package httputil

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

var (
	matchFirstUpper = regexp.MustCompile("(.)([A-Z][a-z]+)")
	matchAllUppers  = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toKebabCase(str string) string {
	kebab := matchFirstUpper.ReplaceAllString(str, "${1}-${2}")
	kebab = matchAllUppers.ReplaceAllString(kebab, "${1}-${2}")

	return strings.ToLower(kebab)
}

func DecodeForm(r *http.Request, dst any) error {
	value := reflect.ValueOf(dst)
	if value.Kind() != reflect.Ptr {
		panic(fmt.Sprintf("want pointer to a struct; got %T", dst))
	}

	s := value.Elem()
	if s.Kind() != reflect.Struct {
		panic(fmt.Sprintf("want pointer to a struct; got %T", dst))
	}

	for i := 0; i < s.NumField(); i++ {
		typeField := s.Type().Field(i)

		tag := typeField.Tag.Get("form")
		if tag == "" {
			tag = toKebabCase(typeField.Name)
		}

		str := r.PostFormValue(tag)
		field := s.Field(i)

		switch typeField.Type.Kind() {
		case reflect.Bool:
			field.SetBool(str == "1" || str == "checked")

		case reflect.Float32:
			value, err := strconv.ParseFloat(str, 32)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetFloat(value)

		case reflect.Float64:
			value, err := strconv.ParseFloat(str, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetFloat(value)

		case reflect.Int8:
			value, err := strconv.ParseInt(str, 10, 8)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Int16:
			value, err := strconv.ParseInt(str, 10, 16)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Int32:
			value, err := strconv.ParseInt(str, 10, 32)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Int64:
			value, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Int:
			value, err := strconv.ParseInt(str, 10, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetInt(value)

		case reflect.Uint8:
			value, err := strconv.ParseUint(str, 10, 8)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.Uint16:
			value, err := strconv.ParseUint(str, 10, 16)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.Uint32:
			value, err := strconv.ParseUint(str, 10, 32)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.Uint64:
			value, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.Uint:
			value, err := strconv.ParseUint(str, 10, 64)
			if err != nil {
				return errors.Tracef(err)
			}

			field.SetUint(value)

		case reflect.String:
			field.SetString(str)

		default:
			panic(fmt.Sprintf("unsupported struct field type %q", typeField.Type.Kind()))
		}
	}

	return nil
}
