package base

import "sync"

const (
	// 最大傳輸單元(Maximum Transmission Unit): 最大封包大小
	MTU int32 = 1500
)

var packetPool *sync.Pool

func InitPacketPool(bufferSize int32) {
	if packetPool == nil {
		packetPool = &sync.Pool{
			New: func() any {
				return NewPacket(bufferSize)
			},
		}
	}
}

func GetPacket() *Packet {
	return packetPool.Get().(*Packet)
}

func PutPacket(packet *Packet) {
	packet.Release()
	packetPool.Put(packet)
}

type Packet struct {
	// 數據緩存
	Data []byte
	// 起始位置(若讀寫一次成功，則基本會是 0)
	Index int32
	// 容量大小
	capacity int32
	// 數據結束位置(若讀寫一次成功，則基本上是數據長度)
	Length int32
	// 讀寫錯誤
	Error *ConnError
}

func NewPacket(bufferSize int32) *Packet {
	p := &Packet{
		Index:    0,
		capacity: bufferSize,
		Data:     make([]byte, bufferSize),
		Length:   0,
		Error:    nil,
	}
	return p
}

func (p *Packet) GetCapacity() int32 {
	return p.capacity
}

func (p *Packet) Release() {
	p.Index = 0
	p.Length = 0
	p.Error = nil
}
