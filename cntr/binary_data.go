package cntr

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
)

type BinaryData struct {
	// 實際數據
	buffer bytes.Buffer
	// 數據位元組順序
	order binary.ByteOrder
}

func NewBinaryData() *BinaryData {
	b := &BinaryData{
		order: binary.LittleEndian,
	}
	return b
}

func LoadBinaryData(data []byte) (*BinaryData, error) {
	b := NewBinaryData()
	err := b.AddRawData(data)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to load data")
	}
	return b, nil
}

func (b *BinaryData) SetOrder(order binary.ByteOrder) {
	b.order = order
}

func (b *BinaryData) GetCapacity() uint32 {
	return uint32(b.buffer.Cap())
}

func (b *BinaryData) GetLength() uint32 {
	return uint32(b.buffer.Len())
}

func (b *BinaryData) Reset() {
	b.buffer.Reset()
}

// ==================================================
// 加入數據
// ==================================================

func (b *BinaryData) AddRawData(data []byte) error {
	if data == nil {
		data = []byte{}
	}
	if len(data) > 0 {
		_, err := b.buffer.Write(data)
		if err != nil {
			return errors.Wrapf(err, "Failed to write data: %+v", data)
		}
	}
	return nil
}

func (b *BinaryData) AddBoolean(data bool) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %+v", data)
	}
	return nil
}

func (b *BinaryData) AddInt8(data int8) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %d", data)
	}
	return nil
}

func (b *BinaryData) AddInt16(data int16) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %d", data)
	}
	return nil
}

func (b *BinaryData) AddInt32(data int32) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %d", data)
	}
	return nil
}

func (b *BinaryData) AddInt64(data int64) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %d", data)
	}
	return nil
}

func (b *BinaryData) AddByte(data byte) error {
	err := b.buffer.WriteByte(data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %d", data)
	}
	return nil
}

func (b *BinaryData) AddUInt16(data uint16) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %d", data)
	}
	return nil
}

func (b *BinaryData) AddUInt32(data uint32) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %d", data)
	}
	return nil
}

func (b *BinaryData) AddUInt64(data uint64) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %d", data)
	}
	return nil
}

func (b *BinaryData) AddFloat32(data float32) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %f", data)
	}
	return nil
}

func (b *BinaryData) AddFloat64(data float64) error {
	err := binary.Write(&b.buffer, b.order, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %f", data)
	}
	return nil
}

func (b *BinaryData) AddString(data string) error {
	err := b.AddByteArray([]byte(data))
	if err != nil {
		return errors.Wrapf(err, "Failed to write data: %s", data)
	}
	return nil
}

func (b *BinaryData) AddByteArray(data []byte) error {
	// 明確處理 nil，將其視為空 slice
	if data == nil {
		data = []byte{}
	}
	length := uint32(len(data))
	err := b.AddUInt32(length)
	if err != nil {
		return errors.Wrapf(err, "Faield to write length of byte array: %d", length)
	}
	// 只有在有數據時才進行寫入
	if length > 0 {
		_, err = b.buffer.Write(data)
		if err != nil {
			return errors.Wrapf(err, "Failed to write data: %+v", data)
		}
	}
	return nil
}

func (b *BinaryData) AddFloat64Array(values []float64) error {
	// 明確處理 nil，將其視為空 slice
	if values == nil {
		values = []float64{}
	}
	length := uint32(len(values))
	err := b.AddUInt32(length)
	if err != nil {
		return errors.Wrapf(err, "Faield to write length of byte array: %d", length)
	}
	for _, value := range values {
		err = b.AddFloat64(value)
		if err != nil {
			return errors.Wrapf(err, "Faield to write float64 data: %f", value)
		}
	}
	return nil
}

func (b *BinaryData) AddMapStringString(data map[string]string) error {
	if data == nil {
		data = make(map[string]string)
	}
	length := uint32(len(data))
	err := b.AddUInt32(length)
	if err != nil {
		return errors.Wrapf(err, "Failed to write length of map: %d", length)
	}
	for k, v := range data {
		err = b.AddString(k)
		if err != nil {
			return errors.Wrapf(err, "Failed to write key of map: %s", k)
		}
		err = b.AddString(v)
		if err != nil {
			return errors.Wrapf(err, "Failed to write value of map: %s", v)
		}
	}
	return nil
}

func (b *BinaryData) AddMapStringByteArray(data map[string][]byte) error {
	if data == nil {
		data = make(map[string][]byte)
	}
	length := uint32(len(data))
	err := b.AddUInt32(length)
	if err != nil {
		return errors.Wrapf(err, "Failed to write length of map: %d", length)
	}
	for k, v := range data {
		err = b.AddString(k)
		if err != nil {
			return errors.Wrapf(err, "Failed to write key of map: %s", k)
		}
		err = b.AddByteArray(v)
		if err != nil {
			return errors.Wrapf(err, "Failed to write value of map: %+v", v)
		}
	}
	return nil
}

// ==================================================
// 插入數據(目前只能插在最前面)
// ==================================================

func (b *BinaryData) InsertInt32(data int32) error {
	err := insertNumber(b, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to insert int32 data: %d", data)
	}
	return nil
}

func (b *BinaryData) InsertUInt32(data uint32) error {
	err := insertNumber(b, data)
	if err != nil {
		return errors.Wrapf(err, "Failed to insert uint32 data: %d", data)
	}
	return nil
}

// ==================================================
// 查看全部的數據
// ==================================================
func (b BinaryData) GetData() []byte {
	return b.buffer.Bytes()
}

// ==================================================
// 取出數據
// ==================================================
func (b *BinaryData) PopRawData() ([]byte, error) {
	data := make([]byte, b.GetLength())
	_, err := b.buffer.Read(data)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read raw data")
	}
	return data, nil
}

func (b *BinaryData) PopBoolean() (bool, error) {
	boolean, err := b.PopByte()
	if err != nil {
		return false, errors.Wrap(err, "Failed to read bool data")
	}
	return boolean == 1, nil
}

func (b *BinaryData) PopInt8() (int8, error) {
	value, err := popNumber[int8](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read int8 data")
	}
	return value, nil
}

func (b *BinaryData) PopInt16() (int16, error) {
	value, err := popNumber[int16](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read int16 data")
	}
	return value, nil
}

func (b *BinaryData) PopInt32() (int32, error) {
	value, err := popNumber[int32](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read int32 data")
	}
	return value, nil
}

func (b *BinaryData) PopInt64() (int64, error) {
	value, err := popNumber[int64](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read int64 data")
	}
	return value, nil
}

func (b *BinaryData) PopByte() (byte, error) {
	value, err := popNumber[uint8](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read uint8 data")
	}
	return value, nil
}

func (b *BinaryData) PopUInt16() (uint16, error) {
	value, err := popNumber[uint16](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read uint16 data")
	}
	return value, nil
}

func (b *BinaryData) PopUInt32() (uint32, error) {
	value, err := popNumber[uint32](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read uint32 data")
	}
	return value, nil
}

func (b *BinaryData) PopUInt64() (uint64, error) {
	value, err := popNumber[uint64](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read uint64 data")
	}
	return value, nil
}

func (b *BinaryData) PopFloat32() (float32, error) {
	value, err := popNumber[float32](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read float32 data")
	}
	return value, nil
}

func (b *BinaryData) PopFloat64() (float64, error) {
	value, err := popNumber[float64](b)
	if err != nil {
		return 0, errors.Wrap(err, "Failed to read float64 data")
	}
	return value, nil
}

func (b *BinaryData) PopMapStringString() (map[string]string, error) {
	result := map[string]string{}
	length, err := b.PopInt32()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read length of map")
	}
	var key, value string
	for i := int32(0); i < length; i++ {
		key, err = b.PopString()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read key of map")
		}
		value, err = b.PopString()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read value of map")
		}
		result[key] = value
	}
	return result, nil
}

func (b *BinaryData) PopMapStringByteArray() (map[string][]byte, error) {
	result := map[string][]byte{}
	length, err := b.PopInt32()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read length of map")
	}
	var key string
	var value []byte
	for i := int32(0); i < length; i++ {
		key, err = b.PopString()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read key of map")
		}
		value, err = b.PopByteArray()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read value of map")
		}
		result[key] = value
	}
	return result, nil
}

func (b *BinaryData) PopString() (string, error) {
	result, err := b.PopByteArray()
	if err != nil {
		return "", errors.Wrap(err, "Faield to read string data")
	}
	return string(result), nil
}

func (b *BinaryData) PopByteArray() ([]byte, error) {
	length, err := b.PopUInt32()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read length of byte array")
	}
	// 明確處理空陣列的情況
	if length == 0 {
		return []byte{}, nil
	}
	result, err := b.FetchByteArray(length)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to fetch byte array")
	}
	return result, nil
}

// 讀取 byte 陣列
func (b *BinaryData) FetchByteArray(length uint32) ([]byte, error) {
	// 如果長度為 0，直接返回空 slice
	if length == 0 {
		return []byte{}, nil
	}
	result := make([]byte, length)
	err := binary.Read(&b.buffer, b.order, result)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read byte array")
	}
	return result, nil
}

func (b *BinaryData) PopFloat64Array() ([]float64, error) {
	length, err := b.PopUInt32()
	if err != nil {
		return nil, errors.Wrap(err, "Failed to read length of byte array")
	}
	// 明確處理空陣列的情況
	if length == 0 {
		return []float64{}, nil
	}
	result := make([]float64, length)
	for i := uint32(0); i < length; i++ {
		result[i], err = b.PopFloat64()
		if err != nil {
			return nil, errors.Wrap(err, "Failed to read data of float64 array")
		}
	}
	return result, nil
}

// ==================================================
// Tools
// ==================================================

func insertNumber[T NumberX](b *BinaryData, v T) error {
	data, err := b.PopRawData()
	fmt.Printf("insertNumber raw data: %+v\n", data)
	if err != nil {
		return errors.Wrap(err, "Failed to pop raw data")
	}
	err = binary.Write(&b.buffer, b.order, v)
	if err != nil {
		return errors.Wrapf(err, "Failed to insert number: %+v", v)
	}
	err = b.AddRawData(data)
	if err != nil {
		return errors.Wrap(err, "Failed to rewrite original data")
	}
	return nil
}

func popNumber[T NumberX](b *BinaryData) (T, error) {
	var v T
	err := binary.Read(&b.buffer, b.order, &v)
	if err != nil {
		return v, errors.Wrap(err, "Failed to read data")
	}
	return v, nil
}
