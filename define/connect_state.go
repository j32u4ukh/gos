package define

type ConnectState int32

const (
	// 未使用
	Unused ConnectState = iota
	// 嘗試連線中
	Connecting
	// 連線中
	Connected
	// 超時斷線
	Timeout
	// 斷線
	Disconnected
	// 重新連線中
	Reconnecting
)
