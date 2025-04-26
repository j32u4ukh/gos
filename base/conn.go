package base

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/cntr"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/log"
	"github.com/j32u4ukh/gos/utils"
)

type Conn struct {
	// 連線物件編號
	id int32
	wg sync.WaitGroup
	// ==================================================
	// 連線結構
	// ==================================================
	// 連線物件
	netConn net.Conn
	// 連線狀態
	state define.ConnectState
	// Handler 中斷用
	cancel context.CancelFunc
	// ==================================================
	// 讀寫結構
	// ==================================================
	// 緩衝長度
	bufferLength int32
	// ========== 讀取 ==========
	// 讀取緩衝
	readBuffer *cntr.BinaryData
	order      binary.ByteOrder
	// 讀取通道
	readCh chan cntr.Void
	// 讀取超時
	readTimeout time.Duration
	// ========== 寫出 ==========
	// 寫出通道
	writeCh chan *Packet
	// ========== 錯誤 ==========
	// 錯誤通道
	errorCh chan *ConnError
	// ========== 關閉連線 ==========
	shouldCloseFunc func() bool
	closeAfter      time.Duration
	closeOnce       sync.Once
}

func NewConn(id int32, size int32) *Conn {
	c := &Conn{
		id:              id,
		netConn:         nil,
		state:           define.Unused,
		bufferLength:    size * define.MTU,
		readCh:          make(chan cntr.Void, size),
		readBuffer:      cntr.NewBinaryData(),
		order:           binary.LittleEndian,
		writeCh:         make(chan *Packet, size),
		readTimeout:     5 * time.Second,
		errorCh:         make(chan *ConnError, size),
		shouldCloseFunc: func() bool { return false },
		closeAfter:      3 * time.Second,
	}
	InitPacketPool(c.bufferLength)
	return c
}

// 取得連線物件編號
func (c *Conn) GetId() int32 {
	return c.id
}

func (c *Conn) SetNetConn(netConn net.Conn) {
	c.netConn = netConn
}

func (c *Conn) SetState(state define.ConnectState) {
	c.state = state
}

func (c *Conn) SetReadTimeout(timeout time.Duration) {
	c.readTimeout = timeout
}

func (c *Conn) SetBinaryOrder(order binary.ByteOrder) {
	c.order = order
	c.readBuffer.SetOrder(order)
}

func (c *Conn) GetBinaryOrder() binary.ByteOrder {
	return c.order
}

func (c *Conn) SetCancelFunc(cancel context.CancelFunc) {
	c.cancel = cancel
}

func (c *Conn) ReadCh() <-chan cntr.Void {
	return c.readCh
}

func (c *Conn) ErrorCh() <-chan *ConnError {
	return c.errorCh
}

// ==================================================
// Handler
// ==================================================
func (c *Conn) Handler(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("(c *Conn) Handler | recover: %+v\n", r)
		}
	}()
	c.wg.Add(2)
	go c.readHandler(ctx)
	go c.writeHandler(ctx)
	c.wg.Wait()
	c.Release()
}

func (c *Conn) readHandler(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recover: %+v\n", r)
		}
		c.wg.Done()
	}()
	for {
		select {
		case <-ctx.Done():
			return
		default:
			c.read()
		}
	}
}

func (c *Conn) read() {
	packet := GetPacket()
	defer PutPacket(packet)
	// 更新斷線時間(NOTE: 若斷線時間與客戶端睡眠時間相同，會變成讀取錯誤，而非 timeout 錯誤，造成誤判)
	err := c.netConn.SetReadDeadline(time.Now().Add(c.readTimeout))
	if err != nil {
		log.Error("DeadlineError: %+v", err)
		return
	}
	// 每次讀取至多長度為 MTU 的數據(Read 為阻塞型函式)
	nRead, err := c.netConn.Read(packet.Data)
	// 封包讀取發生異常
	if err != nil {
		switch eType := err.(type) {
		case net.Error:
			if eType.Timeout() {
				packet.Error = &ConnError{
					Type: ErrTypeTimeout,
					Err:  fmt.Errorf("%s 發生 timeout error.", c.netConn.RemoteAddr()),
				}
			} else {
				packet.Error = &ConnError{
					Type: ErrTypeNetwork,
					Err:  fmt.Errorf("%s 發生 net.Error.", c.netConn.RemoteAddr()),
				}
			}
		default:
			switch err {
			// 沒有數據可讀取，對方已關閉連線
			case io.EOF:
				packet.Error = &ConnError{
					Type: ErrTypeEOF,
					Err:  fmt.Errorf("%s 沒有數據可讀取，對方已關閉連線, Error(%v): %+v", c.netConn.RemoteAddr(), eType, err),
				}
			default:
				packet.Error = &ConnError{
					Type: ErrTypeNetwork,
					Err:  fmt.Errorf("%s 讀取 socket 時發生錯誤, Error(%v): %+v", c.netConn.RemoteAddr(), eType, err),
				}
			}
		}
		c.errorCh <- packet.Error
		return
	}
	c.readBuffer.AddRawData(packet.Data[:nRead])
	// 將讀取到的封包加入通道
	c.readCh <- cntr.NULL
}

func (c *Conn) writeHandler(ctx context.Context) {
	defer c.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Error("recover: %+v", r)
		}
		// 關閉寫入通道，表示不再發送數據
		if err := c.netConn.(*net.TCPConn).CloseWrite(); err != nil {
			log.Error("Error closing write: %+v", err.Error())
		}
	}()
	var packet *Packet
	for {
		select {
		case <-ctx.Done():
			return
		case packet = <-c.writeCh:
			c.write(packet)
			// 如果寫完封包且設定了延遲關閉
			if c.shouldCloseFunc() {
				c.Stop()
			}
		}
	}
}

func (c *Conn) write(packet *Packet) {
	defer PutPacket(packet)
	var nWrite int
	var nWrite32 int32
	var err error
	for packet.Length > 0 {
		// 將數據寫出(Write 為阻塞型函式)
		nWrite, err = c.netConn.Write(packet.Data[packet.Index : packet.Index+packet.Length])
		if err != nil {
			break
		}
		nWrite32 = int32(nWrite)
		packet.Index += nWrite32
		packet.Length -= nWrite32
	}
	if err != nil {
		log.Error("Failed to write packet for conn-%d, err: %+v", c.GetId(), err)
		switch eType := err.(type) {
		case net.Error:
			if eType.Timeout() {
				packet.Error = &ConnError{
					Type: ErrTypeTimeout,
					Err:  fmt.Errorf("%s 發生 timeout error.", c.netConn.RemoteAddr()),
				}
			} else {
				packet.Error = &ConnError{
					Type: ErrTypeNetwork,
					Err:  fmt.Errorf("%s 發生 net.Error.", c.netConn.RemoteAddr()),
				}
			}
		default:
			switch err {
			// 沒有數據可讀取，對方已關閉連線
			case io.EOF:
				packet.Error = &ConnError{
					Type: ErrTypeEOF,
					Err:  fmt.Errorf("%s 沒有數據可讀取，對方已關閉連線, Error(%v): %+v", c.netConn.RemoteAddr(), eType, err),
				}
			default:
				packet.Error = &ConnError{
					Type: ErrTypeNetwork,
					Err:  fmt.Errorf("%s 讀取 socket 時發生錯誤, Error(%v): %+v", c.netConn.RemoteAddr(), eType, err),
				}
			}
		}
		c.errorCh <- packet.Error
	}
}

// ==================================================
// Buffer
// ==================================================

func (c *Conn) GetBuffer() *cntr.BinaryData {
	return c.readBuffer
}

func (c *Conn) GetData() []byte {
	return c.readBuffer.GetData()
}

func (c *Conn) GetReadableLength() int32 {
	return int32(c.readBuffer.GetLength())
}

// ==================================================
// Read / Write
// ==================================================
func (c *Conn) Read(length uint32) ([]byte, error) {
	return c.readBuffer.FetchByteArray(length)
}

func (c *Conn) Write(data []byte) {
	offset := int32(0)
	length := int32(len(data))
	var packet *Packet
	var len int32
	for length > 0 {
		len = utils.Min(length, c.bufferLength)
		packet = GetPacket()
		copy(packet.Data[:len], data[offset:offset+len])
		packet.Length = len
		offset += len
		c.writeCh <- packet
		length -= len
	}
}

// ==================================================
// Status
// ==================================================
func (c *Conn) SetShouldCloseAfterWrite(duration time.Duration, shouldCloseFunc func() bool) {
	c.closeAfter = duration
	c.shouldCloseFunc = shouldCloseFunc
}

func (c *Conn) Stop() {
	if c.cancel != nil {
		c.closeOnce.Do(func() {
			go func() {
				time.Sleep(c.closeAfter)
				c.cancel()
			}()
		})
	}
}

// 清空讀取緩存
func (c *Conn) ResetReadBuffer() {
	c.readBuffer.Reset()
}

func (c *Conn) Release() {
	// 狀態設置為未使用
	c.state = define.Unused
	if c.netConn != nil {
		// 關閉當前連線
		c.netConn.Close()
		// 清空連線物件
		c.netConn = nil
	}
	// 清空讀取緩存
	c.readBuffer.Reset()
	// 清空 WriteCh
	c.clearWriteCh()
}

// 清空 WriteCh
func (c *Conn) clearWriteCh() {
	for {
		select {
		case packet := <-c.writeCh:
			PutPacket(packet)
		default:
			return
		}
	}
}
