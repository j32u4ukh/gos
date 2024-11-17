package gtcp

import (
	"github.com/j32u4ukh/gos/async/gos/base"
)

type TcpContext struct {
	*base.Context
	// 讀取狀態值 | 0: 讀取數據長度, 1: 根據前一階段取得的長度，讀取數據
	State      int8
	HeaderSize int32
	ReadLength int32
}

func NewTcp0() *TcpContext {
	t := &TcpContext{
		State:      0,
		Context:    base.NewContext(64 * 1024),
		HeaderSize: 4,
	}
	t.ReadLength = t.HeaderSize
	return t
}

// 檢查是否滿足：可讀長度 大於 欲讀取長度
func (t *TcpContext) ReadableChecker(buffer []byte, i int32, o int32, length int32) bool {
	return length >= t.ReadLength
}

func (t *TcpContext) Reset() {
	t.Context.Reset()
	t.ReadLength = t.HeaderSize
}
