package openapi

type Operation struct {
	Summary     string
	Description string
	Tags        []string
	Deprecated  bool
}

func Describe[T any](handler T, _ Operation) T {
	return handler
}
