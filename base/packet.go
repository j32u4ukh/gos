package base

const (
	// 最大傳輸單元(Maximum Transmission Unit): 最大封包大小
	MTU int32 = 1500
)

type Packet struct {
	// 數據緩存
	Data []byte
	// 起始位置(若讀寫一次成功，則基本會是 0)
	Index int32
	// 數據結束位置(若讀寫一次成功，則基本上是數據長度)
	Length int32
	// 讀寫錯誤
	Error error
}

func NewPacket() *Packet {
	p := &Packet{
		Index:  0,
		Data:   make([]byte, MTU),
		Length: 0,
		Error:  nil,
	}
	return p
}

func (p *Packet) Release() {
	p.Length = 0
	p.Error = nil
}
