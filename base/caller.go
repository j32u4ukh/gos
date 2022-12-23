package base

import "net"

// goserver 需要傳遞自身的多個函式到 ans 與 ask 當中，為避免傳遞多個函式造成參數量暴增，
// 以及避免直接傳遞 goserver 造成循環引用，因此額外建立此 struct 來乘載多組函式並傳遞到 ans 與 ask 當中
type Caller struct {
	RegisterConn func(int32, net.Conn)
}
