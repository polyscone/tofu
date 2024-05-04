package httpx

import (
	"fmt"
	"mime"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/polyscone/tofu/size"
)

var decodeTimeFormats = []string{
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02T15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02T15:04:05Z07:00",
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	"2006-01-02 15:04Z07:00",
	"2006-01-02T15:04Z07:00",
	"2006-01-02 15:04",
	"2006-01-02T15:04",
	"2006-01-02Z07:00",
	"2006-01-02",
	"2006-01",
	"2006",
	"15:04:05.999999999Z07:00",
	"15:04:05.999999999",
	"15:04:05Z07:00",
	"15:04:05",
	"15:04Z07:00",
	"15:04",
}

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

	for i := range s.NumField() {
		typeField := s.Type().Field(i)

		tagValue := typeField.Tag.Get(tagName)
		field := s.Field(i)

		strs, err := fn(r, typeField.Name, tagValue)
		if err != nil {
			return fmt.Errorf("request value: %w", err)
		}

		var str string
		if len(strs) > 0 {
			str = strs[0]
		}

		switch typ := typeField.Type; typ.Kind() {
		case reflect.Bool:
			compare := typeField.Tag.Get("compare")
			if compare == "" {
				panic(fmt.Sprintf("want `compare` value tag for field %q", typeField.Name))
			}

			field.SetBool(str == compare)

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

			case reflect.TypeOf(time.Time{}):
				if str != "" {
					for _, format := range decodeTimeFormats {
						if t, err := time.ParseInLocation(format, str, time.UTC); err == nil {
							field.Set(reflect.ValueOf(t))

							return nil
						}
					}

					if strings.Contains(str, ".") {
						_sec, _nsec, ok := strings.Cut(str, ".")

						sec, err := strconv.ParseInt(_sec, 10, 64)
						if err != nil {
							return fmt.Errorf("parse time.Time: string value %q is an invalid unix timestamp", str)
						}

						var nsec int64
						if ok {
							var err error
							nsec, err = strconv.ParseInt(_nsec, 10, 64)
							if err != nil {
								return fmt.Errorf("parse time.Time: string value %q is an invalid unix timestamp", str)
							}
						}

						t := time.Unix(sec, nsec)

						field.Set(reflect.ValueOf(t))

						return nil
					}

					return fmt.Errorf("parse time.Time: string value %q is an invalid time format", str)
				}

			default:
				panic(fmt.Sprintf("unsupported struct field type %v", typ))
			}
		}
	}

	return nil
}

func DecodeRequestForm(dst any, r *http.Request) error {
	return DecodeRequest(dst, r, "form", func(r *http.Request, fieldName, tagValue string) ([]string, error) {
		key := tagValue
		if key == "" {
			key = fieldName
		}

		if err := r.ParseForm(); err != nil {
			return nil, fmt.Errorf("parse form: %w", err)
		}

		if r.MultipartForm == nil {
			contentType := r.Header.Get("content-type")
			mediaType, _, err := mime.ParseMediaType(contentType)
			if err == nil && (mediaType == "multipart/form-data" || mediaType == "multipart/mixed") {
				const maxMemory = 32 * size.Megabyte
				if err := r.ParseMultipartForm(maxMemory); err != nil {
					return nil, fmt.Errorf("parse multipart form: %w", err)
				}
			}
		}

		return r.PostForm[key], nil
	})
}

func DecodeRequestQuery(dst any, r *http.Request) error {
	return DecodeRequest(dst, r, "query", func(r *http.Request, fieldName, tagValue string) ([]string, error) {
		key := tagValue
		if key == "" {
			key = fieldName
		}

		return r.URL.Query()[key], nil
	})
}
