package main

import (
	"fmt"
	"reflect"

	"github.com/j32u4ukh/gos/base"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type Protocol struct {
	Kind    byte
	Service uint16
	Data    []byte
}

func (p *Protocol) String() string {
	return fmt.Sprintf("{Kind: %d, Service: %d, Data: %+v}", p.Kind, p.Service, p.Data)
}

func (p *Protocol) Unmarshal(data []byte) {
	v := reflect.ValueOf(p).Elem()
	td := base.LoadTransData(data)

	field := v.FieldByIndex([]int{0})
	field.SetUint(uint64(td.PopByte()))

	field = v.FieldByIndex([]int{1})
	field.SetUint(uint64(td.PopUInt16()))

	field = v.FieldByIndex([]int{2})
	field.SetBytes(td.GetData())
}

type IProtocol interface {
	Marshal() []byte
	Unmarshal(p *Protocol, f func(reflect.Value, *base.TransData))
}

func Marshal(p IProtocol) []byte {
	v := reflect.ValueOf(p).Elem()
	t := reflect.TypeOf(p).Elem()
	var kind string
	td := base.NewTransData()

	field := v.FieldByIndex([]int{0})
	protocol := field.Interface().(*Protocol)
	td.AddByte(protocol.Kind)
	td.AddUInt16(protocol.Service)

	for i := 1; i < t.NumField(); i++ {
		field = v.FieldByIndex([]int{i})
		kind = field.Type().String()

		switch kind {
		case "bool":
			td.AddBoolean(field.Bool())

		case "int8":
			td.AddInt8(int8(field.Int()))

		case "int16":
			td.AddInt16(int16(field.Int()))

		case "int32":
			td.AddInt32(int32(field.Int()))

		case "int64":
			td.AddInt64(field.Int())

		case "uint8":
			td.AddByte(byte(field.Uint()))

		case "uint16":
			td.AddUInt16(uint16(field.Uint()))

		case "uint32":
			td.AddUInt32(uint32(field.Uint()))

		case "uint64":
			td.AddUInt64(field.Uint())

		case "float32":
			td.AddFloat32(float32(field.Float()))

		case "float64":
			td.AddFloat64(field.Float())

		case "string":
			td.AddString(field.String())

		case "protoreflect.ProtoMessage":
			msg := field.Interface().(protoreflect.ProtoMessage)
			bs, _ := proto.Marshal(msg)
			td.AddByteArray(bs)
		}
	}

	return td.GetData()
}

func Unmarshal(ip IProtocol, p *Protocol, f func(reflect.Value, *base.TransData)) {
	td := base.LoadTransData(p.Data)
	v := reflect.ValueOf(ip).Elem()
	t := reflect.TypeOf(ip).Elem()
	var field reflect.Value
	var kind string

	field = v.FieldByIndex([]int{0}).Elem()

	// Protocol.Kind
	k := field.FieldByIndex([]int{0})
	k.SetUint(uint64(p.Kind))

	// Protocol.Service
	s := field.FieldByIndex([]int{1})
	s.SetUint(uint64(p.Service))

	// Protocol.Data
	for i := 1; i < t.NumField(); i++ {
		field = v.FieldByIndex([]int{i})
		kind = field.Type().String()

		switch kind {
		case "bool":
			result := td.PopBoolean()
			field.SetBool(result)
		case "int8":
			result := td.PopInt8()
			field.SetInt(int64(result))
		case "int16":
			result := td.PopInt16()
			field.SetInt(int64(result))
		case "int32":
			result := td.PopInt32()
			field.SetInt(int64(result))
		case "int64":
			result := td.PopInt64()
			field.SetInt(result)
		case "uint8":
			result := td.PopByte()
			field.SetUint(uint64(result))
		case "uint16":
			result := td.PopUInt16()
			field.SetUint(uint64(result))
		case "uint32":
			result := td.PopUInt32()
			field.SetUint(uint64(result))
		case "uint64":
			result := td.PopUInt64()
			field.SetUint(result)
		case "float32":
			result := td.PopFloat32()
			field.SetFloat(float64(result))
		case "float64":
			result := td.PopFloat64()
			field.SetFloat(result)
		case "string":
			s := td.PopString()
			field.SetString(s)
		default:
			if f != nil {
				f(field, td)
			}
		}
	}
}
