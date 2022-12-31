package define

type ConnectState int32

const (
	// 未使用
	Unused ConnectState = iota
	// 嘗試連線中(如果 通訊模式 尚未確定，就傳回嘗試連線中)
	Connecting
	// 連線中
	Connected
	// 超時斷線
	Timeout
	// 斷線
	Disconnect
	// 重新連線中
	Reconnect
)

var ConnectStateString = []string{
	"Unused",
	"Connecting",
	"Connected",
	"Timeout",
	"Disconnect",
	"Reconnect",
}

func (cs ConnectState) String() string {
	return ConnectStateString[cs]
}
