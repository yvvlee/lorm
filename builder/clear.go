package builder

func resetSlice[T any](items []T) []T {
	clear(items)
	return items[:0]
}
