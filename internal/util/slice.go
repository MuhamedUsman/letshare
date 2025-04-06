package util

func ToAnySlice[T any](in []T) []any {
	ret := make([]any, len(in))
	for i, v := range in {
		ret[i] = v
	}
	return ret
}
