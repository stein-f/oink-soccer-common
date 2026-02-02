package lang

func ToPtr[T any](v T) *T {
	return &v
}
