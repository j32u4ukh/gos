package utils

func Min[T int32 | uint32](a, b T) T {
	if a <= b {
		return a
	} else {
		return b
	}
}

func Max[T int32 | uint32](a, b T) T {
	if a >= b {
		return a
	} else {
		return b
	}
}
