package main

import (
	"bytes"
	"fmt"
)

type IData interface {
	GetValue1() int64
	GetValue2() int64
}

type Data struct {
	Value1 int64
	Value2 int64
}

func (d *Data) GetValue1() int64 { return d.Value1 }
func (d *Data) GetValue2() int64 { return d.Value2 }
func (d *Data) GetId() int64     { return d.Value1*10 + d.Value2 }

type DBA[T IData] struct {
	datas map[int64]T
	GetId func(T) int64
}

func NewDBA[T IData]() *DBA[T] {
	d := &DBA[T]{
		datas: make(map[int64]T),
	}
	return d
}

func (d *DBA[T]) Add(t T) {
	id := d.GetId(t)
	d.datas[id] = t
}

func (d *DBA[T]) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	for k, v := range d.datas {
		buffer.WriteString(fmt.Sprintf("{%d: %+v}", k, v))
	}
	buffer.WriteString("}")
	return buffer.String()
}

type DataManager struct {
	*DBA[*Data]
}

func NewDataManager() *DataManager {
	m := &DataManager{
		DBA: NewDBA[*Data](),
	}
	m.DBA.GetId = func(d *Data) int64 {
		return d.GetId()
	}
	return m
}

func main() {
	bs := []byte{123, 34, 105, 110, 100, 101, 120, 34, 58, 51, 44, 34, 109, 115, 103, 34, 58, 34, 71, 69, 84, 32, 124, 32, 47, 97, 98, 99, 47, 103, 101, 116, 34, 125}
	fmt.Println(string(bs))
}
