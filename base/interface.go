package base

type IServer interface {
	ReadFunc(int32, []byte)
}
