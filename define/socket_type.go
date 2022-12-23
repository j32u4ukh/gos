package define

type SocketType byte

const (
	// 前 4 碼為數據長度，後面才是實際要傳的數據
	Tcp0 SocketType = iota
	Http
)
