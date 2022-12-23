package test

import (
	"gos/base"
	"testing"
)

func TestBoolean(t *testing.T) {
	td := base.NewTransData()
	td.AddBoolean(false)
	td.ResetIndex()
	v := td.PopBoolean()

	if v {
		t.Error("fail")
	}
}

func TestInt8(t *testing.T) {
	td := base.NewTransData()
	td.AddInt8(-8)
	td.ResetIndex()
	v := td.PopInt8()

	if v != -8 {
		t.Error("fail")
	}
}

func TestInt16(t *testing.T) {
	td := base.NewTransData()
	td.AddInt16(-16)
	td.ResetIndex()
	v := td.PopInt16()

	if v != -16 {
		t.Error("fail")
	}
}

func TestInt32(t *testing.T) {
	td := base.NewTransData()
	td.AddInt32(-32)
	td.ResetIndex()
	v := td.PopInt32()

	if v != -32 {
		t.Error("fail")
	}
}

func TestInt64(t *testing.T) {
	td := base.NewTransData()
	td.AddInt64(-64)
	td.ResetIndex()
	v := td.PopInt64()

	if v != -64 {
		t.Error("fail")
	}
}

func TestByte(t *testing.T) {
	td := base.NewTransData()
	td.AddByte(8)
	td.ResetIndex()
	v := td.PopByte()

	if v != 8 {
		t.Error("fail")
	}
}

func TestUInt16(t *testing.T) {
	td := base.NewTransData()
	td.AddUInt16(16)
	td.ResetIndex()
	v := td.PopUInt16()

	if v != 16 {
		t.Error("fail")
	}
}

func TestUInt32(t *testing.T) {
	td := base.NewTransData()
	td.AddUInt32(32)
	td.ResetIndex()
	v := td.PopUInt32()

	if v != 32 {
		t.Error("fail")
	}
}

func TestUInt64(t *testing.T) {
	td := base.NewTransData()
	td.AddUInt64(64)
	td.ResetIndex()
	v := td.PopUInt64()

	if v != 64 {
		t.Error("fail")
	}
}

func TestString(t *testing.T) {
	td := base.NewTransData()
	td.AddString("TransData")
	td.ResetIndex()
	v := td.PopString()

	if v != "TransData" {
		t.Error("fail")
	}
}

func TestByteArray(t *testing.T) {
	td := base.NewTransData()
	bs := []byte{9, 5, 2, 7}
	td.AddByteArray(bs)
	td.ResetIndex()
	v := td.PopByteArray()

	for i := 0; i < 4; i++ {
		if v[i] != bs[i] {
			t.Error("fail")
		}
	}
}
