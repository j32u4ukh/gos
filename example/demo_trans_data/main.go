package main

import (
	"fmt"

	"github.com/j32u4ukh/gos/base"
)

func main() {
	// demo1()
	// demo2()
	demo3()
	// demo4()
}

func demo1() {
	td := base.NewTransData()

	td.AddByte(8)
	td.AddUInt16(16)
	td.AddUInt32(32)
	td.AddUInt64(64)
	td.AddString("Hello, world!")
	td.AddBoolean(true)
	td.AddInt8(-8)
	td.AddInt16(-16)
	td.AddInt32(-32)
	td.AddInt64(-64)
	td.AddFloat32(3.2)
	td.AddFloat64(6.4)
	fmt.Printf("data: %v\n", td.GetData())

	fmt.Printf("GetCapacity: %v\n", td.GetCapacity())
	fmt.Printf("GetLength: %v\n", td.GetLength())
	td.ResetIndex()

	fmt.Printf("PopByte: %v\n", td.PopByte())
	fmt.Printf("PopUInt16: %v\n", td.PopUInt16())
	fmt.Printf("PopUInt32: %v\n", td.PopUInt32())
	fmt.Printf("PopUInt64: %v\n", td.PopUInt64())
	fmt.Printf("PopString: %v\n", td.PopString())
	fmt.Printf("PopBoolean: %v\n", td.PopBoolean())
	fmt.Printf("PopInt8: %v\n", td.PopInt8())
	fmt.Printf("PopInt16: %v\n", td.PopInt16())
	fmt.Printf("PopInt32: %v\n", td.PopInt32())
	fmt.Printf("PopInt64: %v\n", td.PopInt64())
	fmt.Printf("PopFloat32: %v\n", td.PopFloat32())
	fmt.Printf("PopFloat64: %v\n", td.PopFloat64())
}

func demo2() {
	td := base.NewTransData()
	td.SetCapacity(1)
	td.AddByte(1)

	// ===== 開始觸發容器大小調整機制 =====
	td.AddByte(8)
	// td.AddUInt16(16)
	td.AddUInt32(32)
	// td.AddUInt64(64)
	// td.AddString("Hello, world!")
	// td.AddBoolean(true)
	// td.AddInt8(-8)
	// td.AddInt16(-16)
	// td.AddInt32(-32)
	// td.AddInt64(-64)
	// td.AddFloat32(3.2)
	// td.AddFloat64(6.4)
	fmt.Printf("當前數據大小: %d, 當前容量大小: %d\n", td.GetLength(), td.GetCapacity())
}

func demo3() {
	td := base.NewTransData()
	td.AddInt32(9527)
	bs := td.FormData()
	fmt.Printf("FormData: %+v\n", bs)
}

func demo4() {
	td := base.NewTransData()
	td.AddByte(0)
	td.AddUInt16(0)
	heartbeat := td.GetData()
	fmt.Printf("heartbeat: %+v\n", heartbeat)
	td.Clear()

	td.AddInt32(97)
	td.AddString("")
	td.AddString("Test")
	td.ResetIndex()
	result := td.GetData()
	fmt.Printf("result: %+v\n", result)
	i32 := td.PopInt32()
	empty := td.PopString()
	str := td.PopString()
	fmt.Printf("i32: %d, empty: %s, str: %s\n", i32, empty, str)
}
