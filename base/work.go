package base

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/gos/define"
)

type WorkState int32

// 工作狀態 -1: 空閒; 0: 完成; 1: 尚未完成; 2: 需寫出數據
const (
	// 空閒
	WORK_FREE WorkState = iota
	// 完成
	WORK_DONE
	// 尚未完成
	WORK_NEED_PROCESS
	// 需寫出數據
	WORK_OUTPUT
)

func (ws WorkState) String() string {
	switch ws {
	case WORK_FREE:
		return "WORK_FREE"
	case WORK_DONE:
		return "WORK_DONE"
	case WORK_NEED_PROCESS:
		return "WORK_PROCESSING"
	case WORK_OUTPUT:
		return "WORK_OUTPUT"
	default:
		return "Unknown WorkState"
	}
}

// 封裝基本連線物件，提供給外部存取
type Work struct {
	// ==================================================
	// ListNode 結構
	// ==================================================
	// Work 唯一碼
	id int32
	// 對應的 Conn id 的 Index
	Index int32
	// 請求發起的時間(若距離實際處理的時間過長，則不處理)
	RequestTime time.Time
	// 下一個工作
	Next *Work
	// ==================================================
	// 工作內容
	// ==================================================

	// 工作狀態 -1: 空閒; 0: 完成; 1: 尚未完成; 2: 需寫出數據
	State WorkState
	// 數據緩衝
	Data []byte
	// 數據長度
	Length int32
	// 數據封裝容器
	Body *TransData
}

func NewWork(id int32) *Work {
	c := &Work{
		id:          id,
		Index:       -2,
		RequestTime: time.Now().UTC(),
		Next:        nil,
		Data:        make([]byte, define.BUFFER_SIZE*define.MTU),
		Body:        NewTransData(),
		State:       WORK_FREE,
	}
	return c
}

func (w *Work) GetId() int32 {
	return w.id
}

func (w *Work) Add(work *Work) {
	curr := w
	for curr.Next != nil {
		curr = curr.Next
	}
	curr.Next = work
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
	w.State = WORK_OUTPUT
}

// 格式化數據寫入緩存
func (w *Work) SendTransData() {
	data := w.Body.FormData()
	w.Body.ResetIndex()
	w.Length = int32(len(data))
	copy(w.Data, data)
	w.State = WORK_OUTPUT
}

func (w *Work) Equals(other *Work) bool {
	return w.id == other.id
}

func (w *Work) Finish() {
	w.State = WORK_DONE
	w.Body.Clear()
}

func (w *Work) Release() {
	w.Index = -2
	w.Next = nil
	w.Length = 0
	w.State = WORK_FREE
	w.Body.Clear()
}

func CheckWorks(works *Work) {
	work := works
	for work != nil {
		fmt.Printf("CheckWorks | %s\n", work)
		work = work.Next
	}
	fmt.Println()
}

func (w *Work) String() string {
	descript := fmt.Sprintf("Work(id: %d, Index: %d, State: %s, requestTime: %+v, next: %+v)",
		w.id,
		w.Index,
		w.State,
		w.RequestTime,
		w.Next != nil,
	)
	return descript
}
