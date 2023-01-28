package utils

import (
	"bytes"
	"fmt"
)

func SliceToString[T byte | int | int32 | int64 | string](elements []T) string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	length := len(elements)
	if length > 0 {
		buffer.WriteString(fmt.Sprintf("%v", elements[0]))
		for i := 1; i < length; i++ {
			buffer.WriteString(fmt.Sprintf(", %v", elements[i]))
		}
	}
	buffer.WriteString("}")
	return buffer.String()
}
