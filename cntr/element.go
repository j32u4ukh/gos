package cntr

var NULL Void = struct{}{}

type Void struct{}

type IntX interface {
	int8 | int16 | int32 | int64
}

type Int interface {
	int | IntX
}

type UIntX interface {
	uint8 | uint16 | uint32 | uint64
}

type UInt interface {
	uint | UIntX
}

type Float interface {
	float32 | float64
}

type NumberX interface {
	IntX | UIntX | Float
}

type Number interface {
	Int | UInt | Float
}

type BinaryElement interface {
	NumberX | string
}

type Element interface {
	Number | string
}
