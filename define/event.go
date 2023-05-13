package define

type EventType int32

const (
	// 連線事件
	OnConnected EventType = iota
	// 準備完成事件
	OnReady
	// 斷線事件
	OnDisconnect
)
