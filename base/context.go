package base

type IContext interface {
	// 取得 Context 唯一碼
	GetId() int32
	// 呼叫此函式的結構需傳入自身指標，只有特定結構有資格使用此函式，否則會返回 nil
	GetWork(any) *Work
	// 設置工作結構 id
	SetWorkId(int32)
	// 取得工作結構 id
	GetWorkId() int32
}
