package httputil

import (
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/polyscone/tofu/internal/pkg/size"
)

type DecodeValueFunc func(r *http.Request, fieldName, tagValue string) ([]string, error)

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
		field := s.Field(i)

		strs, err := fn(r, typeField.Name, tagValue)
		if err != nil {
			return fmt.Errorf("get request value: %w", err)
		}

		var str string
		if len(strs) != 0 {
			str = strs[0]
		}

		switch typ := typeField.Type; typ.Kind() {
		case reflect.Bool:
			field.SetBool(str == "1" || str == "on")

		case reflect.Float32:
			var value float64
			if str != "" {
				value, err = strconv.ParseFloat(str, 32)
				if err != nil {
					return fmt.Errorf("parse float32: %w", err)
				}
			}

			field.SetFloat(value)

		case reflect.Float64:
			var value float64
			if str != "" {
				value, err = strconv.ParseFloat(str, 64)
				if err != nil {
					return fmt.Errorf("parse float64: %w", err)
				}
			}

			field.SetFloat(value)

		case reflect.Int8:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 8)
				if err != nil {
					return fmt.Errorf("parse int8: %w", err)
				}
			}

			field.SetInt(value)

		case reflect.Int16:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 16)
				if err != nil {
					return fmt.Errorf("parse int16: %w", err)
				}
			}

			field.SetInt(value)

		case reflect.Int32:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 32)
				if err != nil {
					return fmt.Errorf("parse int32: %w", err)
				}
			}

			field.SetInt(value)

		case reflect.Int64:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 64)
				if err != nil {
					return fmt.Errorf("parse int64: %w", err)
				}
			}

			field.SetInt(value)

		case reflect.Int:
			var value int64
			if str != "" {
				value, err = strconv.ParseInt(str, 10, 64)
				if err != nil {
					return fmt.Errorf("parse int: %w", err)
				}
			}

			field.SetInt(value)

		case reflect.Uint8:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 8)
				if err != nil {
					return fmt.Errorf("parse uint8: %w", err)
				}
			}

			field.SetUint(value)

		case reflect.Uint16:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 16)
				if err != nil {
					return fmt.Errorf("parse uint16: %w", err)
				}
			}

			field.SetUint(value)

		case reflect.Uint32:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 32)
				if err != nil {
					return fmt.Errorf("parse uint32: %w", err)
				}
			}

			field.SetUint(value)

		case reflect.Uint64:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 64)
				if err != nil {
					return fmt.Errorf("parse uint64: %w", err)
				}
			}

			field.SetUint(value)

		case reflect.Uint:
			var value uint64
			if str != "" {
				value, err = strconv.ParseUint(str, 10, 64)
				if err != nil {
					return fmt.Errorf("parse uint: %w", err)
				}
			}

			field.SetUint(value)

		case reflect.String:
			field.SetString(str)

		default:
			switch typ {
			case reflect.TypeOf([]int(nil)):
				var values []int
				if strs != nil {
					values = make([]int, len(strs))

					for i, str := range strs {
						value, err := strconv.ParseInt(str, 10, 64)
						if err != nil {
							return fmt.Errorf("parse %T element: %w", values, err)
						}

						values[i] = int(value)
					}
				}

				field.Set(reflect.ValueOf(values))

			case reflect.TypeOf([]byte(nil)):
				field.SetBytes([]byte(str))

			case reflect.TypeOf([]string(nil)):
				field.Set(reflect.ValueOf(strs))

			default:
				panic(fmt.Sprintf("unsupported struct field type %v", typ))
			}
		}
	}

	return nil
}

func DecodeForm(dst any, r *http.Request) error {
	return DecodeRequest(dst, r, "form", func(r *http.Request, fieldName, tagValue string) ([]string, error) {
		key := tagValue
		if key == "" {
			key = toKebab(fieldName)
		}

		const maxMemory = 32 * size.Megabyte

		if r.PostForm == nil {
			err := r.ParseMultipartForm(maxMemory)
			if err != nil {
				return nil, fmt.Errorf("parse multipart form: %w", err)
			}
		}

		return r.PostForm[key], nil
	})
}

func DecodeQuery(dst any, r *http.Request) error {
	return DecodeRequest(dst, r, "query", func(r *http.Request, fieldName, tagValue string) ([]string, error) {
		key := tagValue
		if key == "" {
			key = toKebab(fieldName)
		}

		return r.URL.Query()[key], nil
	})
}

var (
	reFirstUpper = regexp.MustCompile("(.)([A-Z][a-z]+)")
	reAllUppers  = regexp.MustCompile("([a-z0-9])([A-Z])")
)

func toKebab(str string) string {
	kebab := reFirstUpper.ReplaceAllString(str, "${1}-${2}")
	kebab = reAllUppers.ReplaceAllString(kebab, "${1}-${2}")

	return strings.ToLower(kebab)
}
