package sqlite

import "reflect"

func Columns(obj any) []string {
	value := reflect.ValueOf(obj)
	elem := value.Elem()
	typ := elem.Type()
	cols := make([]string, 0, elem.NumField())

	for i := 0; i < elem.NumField(); i++ {
		field := typ.Field(i)

		if col := field.Tag.Get("sql"); col != "" {
			cols = append(cols, col)
		}
	}

	return cols
}
