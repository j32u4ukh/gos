package base

import (
	"bytes"
	"fmt"

	"github.com/j32u4ukh/gos/define"
)

type OnEventsFunc map[define.EventType]func(any)
type LoopState byte

const (
	// 迴圈繼續往下執行
	KEEPGOING LoopState = 0
	// 呼叫 continue
	CONTINUE LoopState = 1
	// 呼叫 break
	BREAK LoopState = 2
)

func SliceString(array []byte) string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	if len(array) > 0 {
		buffer.WriteString(fmt.Sprintf("%d", array[0]))
		for _, a := range array {
			buffer.WriteString(", ")
			buffer.WriteString(fmt.Sprintf("%d", a))
		}
	}
	buffer.WriteString("}")
	return buffer.String()
}
