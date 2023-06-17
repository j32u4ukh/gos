package base

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
)

const (
	// max int32 = 2^31，這裡取小於 max int32 的最大二次冪數
	LIMIT_SIZE int32 = 1073741824
)

type TransData struct {
	// 實際數據
	data []byte
	// 讀寫用的索引值
	index int32
	// 數據實際長度
	length int32
	// 容器大小(預設為 1024)
	capacity int32
	// 數據位元組順序
	order binary.ByteOrder
	// 用於暫存數據，避免反覆宣告變數
	temp1 int32
	temp2 int32
}

func NewTransData() *TransData {
	td := &TransData{
		index:    0,
		length:   0,
		capacity: 1024,
		order:    binary.LittleEndian,
	}
	td.data = make([]byte, td.capacity)
	return td
}

func LoadTransData(data []byte) *TransData {
	size := ceilSquare(int32(len(data)))
	td := NewTransData()
	td.SetCapacity(size)
	// TODO: add data into td
	return td
}

func (t *TransData) SetCapacity(capacity int32) {
	t.capacity = ceilSquare(capacity)
	data := make([]byte, t.capacity)

	if t.length > 0 {
		copy(data[:t.length], t.data[:t.length])
	}

	t.data = data
}

func (t *TransData) SetOrder(order binary.ByteOrder) {
	t.order = order
}

func (t *TransData) GetCapacity() int32 {
	return int32(len(t.data))
}

func (t *TransData) GetLength() int32 {
	return t.length
}

func (t *TransData) ResetIndex() {
	t.index = 0
}

func (t *TransData) Clear() {
	t.index = 0
	t.length = 0
}

// 取得格式化數據
func (t *TransData) FormData() []byte {
	t.InsertInt32(t.length)
	t.ResetIndex()
	result := make([]byte, t.length)
	copy(result, t.data[:t.length])
	return result
}

// ==================================================
// 加入數據
// ==================================================

func addDatas(t *TransData, bs []byte) {
	t.temp1 = int32(len(bs))

	// 若新增數據後將超出容量
	if t.index+t.temp1 >= t.capacity {
		// fmt.Printf("[TransData] addDatas | 更新容器大小, 原始容量: %d\n", t.capacity)
		// 更新容器大小
		t.SetCapacity(t.index + t.temp1)
		// fmt.Printf("[TransData] addDatas | 更新容器大小, 更新後容量: %d\n", t.capacity)
	}

	// 寫入數據
	copy(t.data[t.index:t.index+t.temp1], bs)

	// 更新容器屬性
	t.index += t.temp1
	t.length += t.temp1
	// fmt.Printf("[TransData] addDatas | 更新容器屬性, t.temp1: %d, t.index: %d, t.length: %d\n", t.temp1, t.index, t.length)
}

func addData(t *TransData, b byte) {
	if t.index == t.capacity {
		// 更新容器大小
		// fmt.Printf("[TransData] addData | 更新容器屬性 | t.index: %d, t.capacity: %d\n", t.index, t.capacity)
		t.SetCapacity(t.capacity + 1)
	}

	// 寫入數據
	t.data[t.index] = b

	// 更新容器屬性
	t.index += 1
	t.length += 1
}

func (t *TransData) AddRawData(v []byte) {
	addDatas(t, v)
}

func (t *TransData) AddBoolean(v bool) {
	if v {
		addData(t, 1)
	} else {
		addData(t, 0)
	}
}

func addNumber[T int8 | int16 | int32 | int64 | uint16 | uint32 | uint64 | float32 | float64](t *TransData, v T) {
	bs := NumberToBytes(v, t.order)
	addDatas(t, bs)
}

func (t *TransData) AddInt8(v int8) {
	addNumber(t, v)
}

func (t *TransData) AddInt16(v int16) {
	addNumber(t, v)
}

func (t *TransData) AddInt32(v int32) {
	addNumber(t, v)
}

func (t *TransData) AddInt64(v int64) {
	addNumber(t, v)
}

func (t *TransData) AddByte(v byte) {
	addData(t, v)
}

func (t *TransData) AddUInt16(v uint16) {
	addNumber(t, v)
}

func (t *TransData) AddUInt32(v uint32) {
	addNumber(t, v)
}

func (t *TransData) AddUInt64(v uint64) {
	addNumber(t, v)
}

func (t *TransData) AddFloat32(v float32) {
	addNumber(t, v)
}

func (t *TransData) AddFloat64(v float64) {
	addNumber(t, v)
}

func (t *TransData) AddJson(v map[string]string) {
	bs, err := json.Marshal(v)
	if err != nil {
		v = map[string]string{
			"error": fmt.Sprintf("%+v", err),
		}
		bs, _ = json.Marshal(v)
	}
	t.AddByteArray(bs)
}

func (t *TransData) AddString(v string) {
	t.AddByteArray([]byte(v))
}

func (t *TransData) AddByteArray(v []byte) {
	t.temp2 = int32(len(v))
	t.AddInt32(t.temp2)
	addDatas(t, v)
}

// ==================================================
// 插入數據(目前只能插在最前面)
// ==================================================

func insertNumber[T int8 | int16 | int32 | int64 | uint16 | uint32 | uint64 | float32 | float64](t *TransData, v T, bit int32) {
	// 將原始數據往後平移 bit 個 byte
	copy(t.data[bit:t.length+bit], t.data[:t.length])
	// 將數據寫入最前面的 bit 個 byte
	copy(t.data[:bit], NumberToBytes(v, t.order))
	// 更新長度
	t.length += bit
	// 新的讀寫索引值指向最後的位置
	t.index = t.length
}

func (t *TransData) InsertInt32(v int32) {
	insertNumber(t, v, 4)
}

// ==================================================
// 取出數據
// type 1: GetXXX -> ... -> GetXXX -> 數據讀完(t.index == t.length) -> Clear(清空內存數據)
// type 2: GetXXX -> ... -> GetXXX -> 讀取部分數據 -> ResetIndex(重置讀寫索引值) -> GetXXX(再次從最前面開始讀取)
// ==================================================

// 取出全部的數據
func (t *TransData) GetData() []byte {
	result := make([]byte, t.length)
	copy(result, t.data[:t.length])
	return result
}

func (t *TransData) PopBoolean() bool {
	b := t.PopByte()
	return b == 1
}

func popNumber[T int8 | int16 | int32 | int64 | uint16 | uint32 | uint64 | float32 | float64](t *TransData, bit byte) T {
	result := BytesToNumber[T](t.data[t.index:t.index+int32(bit)], t.order)
	t.index += int32(bit)
	return result
}

func (t *TransData) PopInt8() int8 {
	return popNumber[int8](t, 1)
}

func (t *TransData) PopInt16() int16 {
	return popNumber[int16](t, 2)
}

func (t *TransData) PopInt32() int32 {
	return popNumber[int32](t, 4)
}

func (t *TransData) PopInt64() int64 {
	return popNumber[int64](t, 8)
}

func (t *TransData) PopByte() byte {
	result := t.data[t.index]
	t.index += 1
	return result
}

func (t *TransData) PopUInt16() uint16 {
	return popNumber[uint16](t, 2)
}

func (t *TransData) PopUInt32() uint32 {
	return popNumber[uint32](t, 4)
}

func (t *TransData) PopUInt64() uint64 {
	return popNumber[uint64](t, 8)
}

func (t *TransData) PopFloat32() float32 {
	return popNumber[float32](t, 4)
}

func (t *TransData) PopFloat64() float64 {
	return popNumber[float64](t, 8)
}

func (t *TransData) PopJson() map[string]string {
	result := map[string]string{}
	bs := t.PopByteArray()
	err := json.Unmarshal(bs, &result)
	if err != nil {
		json.Unmarshal([]byte(fmt.Sprintf("{\"error\": \"%v\"}", err)), &result)
	}
	return result
}

func (t *TransData) PopString() string {
	result := string(t.PopByteArray())
	return result
}

func (t *TransData) PopByteArray() []byte {
	t.temp1 = t.PopInt32()
	result := make([]byte, t.temp1)
	copy(result, t.data[t.index:t.index+t.temp1])
	t.index += t.temp1
	return result
}

// ==================================================
// Tools
// ==================================================

// 返回大於等於 value 但不大於 LIMIT_SIZE 的二次冪數
func ceilSquare(value int32) int32 {
	if value >= LIMIT_SIZE {
		return LIMIT_SIZE
	}

	temp := value - 1
	temp |= temp >> 1
	temp |= temp >> 2
	temp |= temp >> 4
	temp |= temp >> 8
	temp |= temp >> 16

	if temp < 0 {
		return 1
	} else {
		return temp + 1
	}
}
