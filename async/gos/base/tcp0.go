package base

type Tcp0 struct {
	// 讀取狀態值 | 0: 讀取數據長度, 1: 根據前一階段取得的長度，讀取數據
	State      int8
	HeaderSize int32
	ReadLength int32
}

func NewTcp0() *Tcp0 {
	t := &Tcp0{
		State:      0,
		HeaderSize: 4,
	}
	t.ReadLength = t.HeaderSize
	return t
}

func (t *Tcp0) ResetReadLength() {
	t.State = 0
	t.ReadLength = t.HeaderSize
}

// 檢查是否滿足：可讀長度 大於 欲讀取長度
func (t *Tcp0) ReadableChecker(buffer *[]byte, i int32, o int32, length int32) bool {
	return length >= t.ReadLength
}
