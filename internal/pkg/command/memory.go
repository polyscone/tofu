package command

import (
	"context"
	"fmt"
	"reflect"

	"github.com/polyscone/tofu/internal/pkg/errors"
)

// MemoryBus implements an in-memory command bus.
type MemoryBus struct {
	handlers map[string]reflect.Value
}

// NewMemoryBus returns a new in-memory command bus.
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{handlers: make(map[string]reflect.Value)}
}

// Register will register a command handler keyed by the
// command parameter's type.
func (mb *MemoryBus) Register(handler Handler) {
	handlerFuncType := reflect.TypeOf(handler)

	nIn := handlerFuncType.NumIn()
	if want := 2; want != nIn {
		panic(fmt.Sprintf("handler must have %v parameters (context, command); got %v", want, nIn))
	}

	in0 := handlerFuncType.In(0)
	if !in0.Implements(reflect.TypeOf((*context.Context)(nil)).Elem()) {
		panic(fmt.Sprintf("handler must specify context.Context as the first parameter; got %v", in0))
	}

	nOut := handlerFuncType.NumOut()
	if max, got := 2, nOut; max < got {
		panic(fmt.Sprintf("handler must have no more than %v returns (value, error); got %v", max, got))
	}

	if nOut == 2 {
		outLast := handlerFuncType.Out(nOut - 1)
		if !outLast.Implements(reflect.TypeOf((*error)(nil)).Elem()) {
			panic(fmt.Sprintf("handler must specify error as the last return in a pair; got %v", outLast))
		}
	}

	key := handlerKey(handlerFuncType.In(1))
	if _, ok := mb.handlers[key]; ok {
		panic(fmt.Sprintf("duplicate handler registered for %v (command pointers are considered in duplicate checks)", key))
	}

	mb.handlers[key] = reflect.ValueOf(handler)
}

// Dispatch will dispatch the given command to any matching handler that accepts
// the command's type.
// It panics if no handler can be found.
func (mb *MemoryBus) Dispatch(ctx context.Context, cmd Command) (any, error) {
	key := handlerKey(reflect.TypeOf(cmd))
	handler, ok := mb.handlers[key]
	if !ok {
		panic(fmt.Sprintf("no handler registered for %v", key))
	}

	rets := handler.Call([]reflect.Value{
		reflect.ValueOf(ctx),
		reflect.ValueOf(cmd),
	})

	var res any
	var err error
	switch len(rets) {
	case 1:
		out := rets[0].Interface()
		if v, ok := out.(error); ok {
			err = v
		} else {
			res = out
		}

	case 2:
		res = rets[0].Interface()
		err, _ = rets[1].Interface().(error)
	}

	return res, errors.Tracef(err)
}

func handlerKey(typ reflect.Type) string {
	var prefix string
	for typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
		prefix += "*"
	}

	return prefix + typ.PkgPath() + "." + typ.Name()
}
