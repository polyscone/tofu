package pointer

func From[T any](value T) *T {
	return &value
}
