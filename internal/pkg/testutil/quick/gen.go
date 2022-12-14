package quick

import (
	"fmt"
	"math"
	"math/rand"
	"reflect"
	"strings"
)

const defaultSize = 100

type Generator interface {
	Generate(rand *rand.Rand) any
}

type Invalidator interface {
	Invalidate(rand *rand.Rand, value any) any
}

type Invalid[T Invalidator] struct {
	Wrapped T
}

func (i Invalid[T]) Unwrap() T {
	return i.Wrapped
}

func genString(rand *rand.Rand, size int) string {
	codePoints := make([]rune, size)
	for j := 0; j < size; j++ {
		codePoints[j] = rune(rand.Intn(0x10ffff))
	}

	return string(codePoints)
}

func genFloat(rand *rand.Rand, bits int) float64 {
	if bits != 32 && bits != 64 {
		panic("bits must be 32 or 64")
	}

	max := math.MaxFloat64
	if bits == 32 {
		max = math.MaxFloat32
	}
	f := rand.Float64() * max
	if rand.Int()&1 == 1 {
		f = -f
	}

	return f
}

func genValue(rand *rand.Rand, typ reflect.Type, size int) (reflect.Value, bool) {
	if typ.Kind() == reflect.Struct && strings.HasPrefix(typ.String(), "quick.Invalid[") {
		val := reflect.New(typ).Elem()
		field := val.FieldByName("Wrapped")

		if g, ok := field.Interface().(Generator); ok {
			value := g.Generate(rand)

			if i, ok := field.Interface().(Invalidator); ok {
				value = i.Invalidate(rand, value)
			} else {
				panic(fmt.Sprintf("type %v does not implement Invalidator", field.Type()))
			}

			field.Set(reflect.ValueOf(value))

			return val, true
		}
	}

	if g, ok := reflect.Zero(typ).Interface().(Generator); ok {
		value := g.Generate(rand)

		return reflect.ValueOf(value), true
	}

	val := reflect.New(typ).Elem()
	switch typ.Kind() {
	case reflect.Bool:
		val.SetBool(rand.Int()&1 == 0)

	case reflect.Float32:
		val.SetFloat(genFloat(rand, 32))

	case reflect.Float64:
		val.SetFloat(genFloat(rand, 64))

	case reflect.Int16:
		val.SetInt(int64(rand.Uint64()))

	case reflect.Int32:
		val.SetInt(int64(rand.Uint64()))

	case reflect.Int64:
		val.SetInt(int64(rand.Uint64()))

	case reflect.Int8:
		val.SetInt(int64(rand.Uint64()))

	case reflect.Int:
		val.SetInt(int64(rand.Uint64()))

	case reflect.Uint16:
		val.SetUint(rand.Uint64())

	case reflect.Uint32:
		val.SetUint(rand.Uint64())

	case reflect.Uint64:
		val.SetUint(rand.Uint64())

	case reflect.Uint8:
		val.SetUint(rand.Uint64())

	case reflect.Uint:
		val.SetUint(rand.Uint64())

	case reflect.Uintptr:
		val.SetUint(rand.Uint64())

	case reflect.String:
		size := rand.Intn(size)

		val.SetString(genString(rand, size))

	case reflect.Pointer:
		if rand.Intn(size) == 0 {
			// Nil pointer
			val.Set(reflect.Zero(typ))
		} else {
			elem, ok := genValue(rand, typ.Elem(), size)
			if !ok {
				return val, false
			}

			val.Set(reflect.New(typ.Elem()))
			val.Elem().Set(elem)
		}

	default:
		return val, false
	}

	return val, true
}

func genArgs(rand *rand.Rand, fType reflect.Type) []reflect.Value {
	args := make([]reflect.Value, fType.NumIn())
	for i := range args {
		typ := fType.In(i)
		val, ok := genValue(rand, typ, defaultSize)
		if !ok {
			panic(fmt.Sprintf("cannot generate value of type %v for argument %v", typ, i))
		}

		args[i] = val
	}

	return args
}
