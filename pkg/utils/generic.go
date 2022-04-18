package utils

func mapSlice[A any, B any](input []A, mapper func(A) B) (output []B) {
	output = make([]B, len(input))
	for i, value := range input {
		output[i] = mapper(value)
	}
	return
}
