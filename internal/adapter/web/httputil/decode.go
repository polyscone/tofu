package httputil

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"

	"github.com/polyscone/tofu/internal/pkg/casing"
	"github.com/polyscone/tofu/internal/pkg/errors"
)

type DecodeValueFunc func(r *http.Request, fieldName, tagValue string) string

func DecodeRequest(dst any, r *http.Request, tagName string, fn DecodeValueFunc) error {
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

		tagValue := typeField.Tag.Get(tagName)
		str := fn(r, typeField.Name, tagValue)
		field := s.Field(i)

		var err error
		switch typ := typeField.Type; typ.Kind() {
		case reflect.Bool:
			field.SetBool(str == "1" || str == "checked")

		case reflect.Float32:
			var value float64
			if str != "" {
				value, err = strconv.ParseFloat(str, 32)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetFloat(value)

		case reflect.Float64:
			var value float64
			if str != "" {
				value, err = strconv.ParseFloat(str, 64)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetFloat(value)

		case reflect.Int8:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 8)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetInt(value)

		case reflect.Int16:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 16)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetInt(value)

		case reflect.Int32:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 32)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetInt(value)

		case reflect.Int64:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 64)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetInt(value)

		case reflect.Int:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 64)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetInt(value)

		case reflect.Uint8:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 8)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetUint(value)

		case reflect.Uint16:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 16)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetUint(value)

		case reflect.Uint32:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 32)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetUint(value)

		case reflect.Uint64:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 64)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetUint(value)

		case reflect.Uint:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 64)
				if err != nil {
					return errors.Tracef(err)
				}
			}

			field.SetUint(value)

		case reflect.String:
			field.SetString(str)

		default:
			if typ == reflect.TypeOf([]byte(nil)) {
				field.SetBytes([]byte(str))
			} else {
				panic(fmt.Sprintf("unsupported struct field type %v", typ))
			}
		}
	}

	return nil
}

func DecodeForm(dst any, r *http.Request) error {
	return DecodeRequest(dst, r, "form", func(r *http.Request, fieldName, tagValue string) string {
		if tagValue == "" {
			tagValue = casing.ToKebab(fieldName)
		}

		return r.PostFormValue(tagValue)
	})
}

func DecodeQuery(dst any, r *http.Request) error {
	return DecodeRequest(dst, r, "query", func(r *http.Request, fieldName, tagValue string) string {
		if tagValue == "" {
			tagValue = casing.ToKebab(fieldName)
		}

		return r.URL.Query().Get(tagValue)
	})
}
