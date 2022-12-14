package quick

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"
	"testing"
	"time"
)

type CheckFunc any

func CheckN(t *testing.T, n int, f CheckFunc) {
	t.Helper()

	if n < 0 {
		panic("check iterations must be greater than zero")
	}

	fType := reflect.TypeOf(f)

	nOut := fType.NumOut()
	if want := 1; want != nOut {
		panic(fmt.Sprintf("check function must have %v return (bool); got %v", want, nOut))
	}

	out0 := fType.Out(0)
	if out0.Kind() != reflect.Bool {
		panic(fmt.Sprintf("check function must return a bool; got %v", out0))
	}

	rand := rand.New(rand.NewSource(time.Now().UnixNano()))
	fVal := reflect.ValueOf(f)

	for i := 0; i < n; i++ {
		args := genArgs(rand, fType)

		if !fVal.Call(args)[0].Bool() {
			strs := make([]string, len(args))
			for i, arg := range args {
				iface := arg.Interface()

				if arg.Kind() == reflect.Struct && strings.HasPrefix(arg.Type().String(), "quick.Invalid[") {
					field := arg.FieldByName("Wrapped")
					iface = field.Interface()
				}

				strs[i] = fmt.Sprintf("#%v (%T): %#v", i, iface, iface)
			}

			t.Errorf("failed with inputs:\n%v", strings.Join(strs, "\n"))

			return
		}
	}
}

func Check(t *testing.T, check CheckFunc) {
	t.Helper()

	CheckN(t, 100, check)
}
