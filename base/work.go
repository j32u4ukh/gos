package base

import (
	"fmt"
	"gos/define"
	"time"
)

// 封裝基本連線物件，提供給外部存取
type Work struct {
	// ==================================================
	// ListNode 結構
	// ==================================================
	// Work 唯一碼
	id int32
	// 對應的 Connector 的 Index
	Index int32
	// 請求發起的時間(若距離實際處理的時間過長，則不處理)
	RequestTime time.Time
	// 下一個工作
	Next *Work
	// ==================================================
	// 工作內容
	// ==================================================
	// 工作狀態 -1: 空閒; 0: 完成; 1: 尚未完成; 2: 需寫出數據
	State int32
	// 數據緩衝
	Data []byte
	// 數據長度
	Length int32
	// 數據封裝容器
	Body *TransData
	*Protocol
}

func NewWork(id int32) *Work {
	c := &Work{
		id:          id,
		Index:       -1,
		RequestTime: time.Now().UTC(),
		Next:        nil,
		Data:        make([]byte, define.BUFFER_SIZE*define.MTU),
		Body:        NewTransData(),
		Protocol:    &Protocol{Data: []byte{}},
		State:       -1,
	}
	return c
}

func (w *Work) String() string {
	descript := fmt.Sprintf("Work(id: %d, Index: %d, state: %d, requestTime: %+v, next: %+v)",
		w.id,
		w.Index,
		w.State,
		w.RequestTime,
		w.Next != nil,
	)
	return descript
}

func (w *Work) Add(work *Work) {
	curr := w
	for curr.Next != nil {
		curr = curr.Next
	}
	curr.Next = work
}

func (w *Work) getLength() int32 {
	curr := w
	count := 1
	for curr.Next != nil {
		count += 1
		curr = curr.Next
	}
	return int32(count)
}

func (w *Work) Read() []byte {
	return w.Data
}

// 原始數據寫入緩存
func (w *Work) Send() {
	data := w.Body.GetData()
	w.Length = int32(len(data))
	copy(w.Data[:w.Length], data)
	w.Body.ResetIndex()
	w.State = 2
}

// 格式化數據寫入緩存
func (w *Work) SendTransData() {
	data := w.Body.FormData()
	w.Length = int32(len(data))
	w.Body.ResetIndex()
	copy(w.Data, data)
	w.State = 2
}

func (w *Work) SendProtocol(p IProtocol) {
	w.Body.Clear()
	bs := p.Marshal()
	w.Body.AddRawData(bs)
	w.Body.ResetIndex()
	w.State = 2
}

func (w *Work) ReadProtocol(p IProtocol) {
	p.Unmarshal(w.Protocol, nil)
	w.Body.Clear()
}

func (w *Work) Equals(other *Work) bool {
	return w.id == other.id
}

func (w *Work) Finish() {
	w.State = 0
	w.Body.Clear()
}

func (w *Work) Release() {
	w.Index = -1
	w.Next = nil
	w.Length = 0
	w.State = -1
	w.Body.Clear()
	w.Protocol.Data = []byte{}
}
