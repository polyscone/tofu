package command

import "context"

// Handler represents any function that is capable of handling
// dispatched commands.
// The underlying type is set to any to allow for any signatures, but a handler
// should be of the signature func(context.Context, <command>) [([T,] error)].
// That is, it should accept a context and a command type to handle, and can
// return either nothing, some data, an error, or some data and an error.
type Handler any

// Command represents any data that should be considered a command.
// This can be anything from simple primitives to structures.
type Command any

// Bus defines a command bus interface that allows for the registration of
// command handlers and the dispatch of commands for those handlers.
type Bus interface {
	Register(handler Handler)
	Dispatch(ctx context.Context, cmd Command) (any, error)
}
