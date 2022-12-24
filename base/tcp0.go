package base

type Tcp0 struct {
	State      int8
	headerSize int32
	ReadLength int32
}

func NewTcp0() *Tcp0 {
	t := &Tcp0{
		State:      0,
		headerSize: 4,
	}
	t.ReadLength = t.headerSize
	return t
}

func (t *Tcp0) ResetReadLength() {
	t.State = 0
	t.ReadLength = t.headerSize
}

// 檢查是否滿足：可讀長度 大於 欲讀取長度
func (t *Tcp0) ReadableChecker(buffer *[]byte, i int32, o int32, length int32) bool {
	return length >= t.ReadLength
}
