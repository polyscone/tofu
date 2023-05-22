package sqlite

import "reflect"

var dummyAddr any

func ScanInto(obj any, cols []string) []any {
	if len(cols) == 0 {
		return nil
	}

	var addrs []any

	value := reflect.ValueOf(obj)
	elem := value.Elem()
	typ := elem.Type()

	for _, col := range cols {
		var found bool

		for i := 0; i < elem.NumField(); i++ {
			if field := typ.Field(i); field.Tag.Get("sql") != col {
				continue
			}

			field := elem.Field(i)
			if field.CanAddr() {
				field = field.Addr()
			}
			addrs = append(addrs, field.Interface())
			found = true
		}

		if !found {
			addrs = append(addrs, &dummyAddr)
		}
	}

	return addrs
}
