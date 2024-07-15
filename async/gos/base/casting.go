package base

import (
	"bytes"
	"encoding/binary"
)

// binary.ByteOrder
// - binary.BigEndian    7 -> [0 0 0 7]
// - binary.LittleEndian 7 -> [7 0 0 0]

// ===== 轉 byte 陣列 =====
// 數字 轉 byte 陣列
func NumberToBytes[T int8 | int16 | int32 | int64 | uint16 | uint32 | uint64 | float32 | float64](v T, order binary.ByteOrder) []byte {
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, order, v)
	return bytesBuffer.Bytes()
}

// ===== byte 陣列轉回原始數值 =====

func BytesToNumber[T int8 | int16 | int32 | int64 | uint16 | uint32 | uint64 | float32 | float64](b []byte, order binary.ByteOrder) T {
	var result T
	buffer := bytes.NewBuffer(b)
	binary.Read(buffer, order, &result)
	return result
}

// byte 陣列 轉 uint16
func BytesToUInt16(b []byte, order binary.ByteOrder) uint16 {
	var v uint16
	buffer := bytes.NewBuffer(b)
	binary.Read(buffer, order, &v)
	return v
}

// byte 陣列 轉 int32
func BytesToInt32(b []byte, order binary.ByteOrder) int32 {
	var v int32
	buffer := bytes.NewBuffer(b)
	binary.Read(buffer, order, &v)
	return v
}

// byte 陣列 轉 int64
func BytesToInt64(b []byte, order binary.ByteOrder) int64 {
	var v int64
	buffer := bytes.NewBuffer(b)
	binary.Read(buffer, order, &v)
	return v
}
