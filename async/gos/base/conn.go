package base

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/j32u4ukh/cntr"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils/log"
	"github.com/pkg/errors"
)

type ConnMode byte

const (
	CLOSE     ConnMode = 0
	KEEPALIVE ConnMode = 1
)

type ConnBuffer struct {
	net.Conn
	Index int32
}

type Conn struct {
	// 連線物件編號
	id int32
	wg sync.WaitGroup
	// ==================================================
	// 連線結構
	// ==================================================
	// 連線物件
	NetConn net.Conn
	// 連線狀態
	State define.ConnectState
	// Handler 中斷用
	cancel context.CancelFunc
	// ==================================================
	// 讀寫結構
	// ==================================================
	// 緩衝長度
	BufferLength int32
	// ========== 讀取 ==========
	// 讀取緩衝
	readBuffer []byte
	// 讀取輸入索引值(從這個位置開始往後寫入)
	readInput int32
	// 讀取輸出索引值(從這個位置開始往後取出數據，最多到 readInput)
	readOutput int32
	// 可讀取長度(readInput ~ readOutput 之間的數據量)
	ReadableLength int32
	// 讀取封包通道
	ReadCh chan cntr.Void
	// 讀取超時
	ReadTimeout time.Duration
	// ========== 寫出 ==========
	WriteCh chan *Packet
	// 可寫出長度(writeInput ~ writeOutput 之間的數據量)
	WritableLength int32
	// ==================================================
	// 暫存變數(避免重複宣告變數)
	// ==================================================
	writeErr error
}

func NewConn(id int32, size int32) *Conn {
	c := &Conn{
		id:             id,
		NetConn:        nil,
		State:          define.Unused,
		BufferLength:   size * define.MTU,
		readInput:      0,
		readOutput:     0,
		ReadableLength: 0,
		ReadCh:         make(chan cntr.Void, size),
		readBuffer:     make([]byte, MTU),
		WriteCh:        make(chan *Packet, size),
		WritableLength: 0,
		ReadTimeout:    5 * time.Second,
	}
	c.readBuffer = make([]byte, c.BufferLength)
	return c
}

// 取得連線物件編號
func (c *Conn) GetId() int32 {
	return c.id
}

func (c *Conn) SetReadTimeout(timeout time.Duration) {
	c.ReadTimeout = timeout
}

func (c *Conn) Handler(ctx context.Context) {
	var handlerCtx context.Context
	handlerCtx, c.cancel = context.WithCancel(ctx)
	defer c.cancel()
	c.wg.Add(2)
	go c.readHandler(handlerCtx)
	go c.writeHandler(handlerCtx)
	c.Release()
	c.wg.Wait()
}

func (c *Conn) readHandler(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("recover: %+v\n", r)
		}
		c.wg.Done()
	}()
	var packet *Packet
	var nRead int
	var err error
	for {
		select {
		case <-ctx.Done():
			return
		default:
			packet = GetPacket()
			// 每次讀取至多長度為 MTU 的數據(Read 為阻塞型函式)
			nRead, err = c.NetConn.Read(packet.Data)
			// 封包讀取發生異常
			if err != nil {
				switch eType := err.(type) {
				case net.Error:
					if eType.Timeout() {
						log.Error("%s 發生 timeout error.", c.NetConn.RemoteAddr())
					} else {
						log.Error("%s 發生 net.Error.", c.NetConn.RemoteAddr())
					}
				default:
					switch err {
					// 沒有數據可讀取，對方已關閉連線
					case io.EOF:
						log.Warn("%s 沒有數據可讀取，對方已關閉連線\nError(%v): %+v", c.NetConn.RemoteAddr(), eType, err)
					default:
						log.Error("%s 讀取 socket 時發生錯誤, Error(%v): %+v", c.NetConn.RemoteAddr(), eType, err)
					}
				}
				return
			}
			// utils.Debug("readIdx: %d, nRead: %d", c.readIdx, c.nRead)
			packet.Length = int32(nRead)
			c.SetReadBuffer(packet)
			// 更新斷線時間(NOTE: 若斷線時間與客戶端睡眠時間相同，會變成讀取錯誤，而非 timeout 錯誤，造成誤判)
			err = c.NetConn.SetReadDeadline(time.Now().Add(c.ReadTimeout))
			if err != nil {
				log.Error("DeadlineError: %+v", err)
				return
			}
			// 將讀取到的封包加入通道
			c.ReadCh <- cntr.NULL
		}
	}
}

// 讀取封包數據，並寫入 readBuffer
func (c *Conn) SetReadBuffer(packet *Packet) {
	defer PutPacket(packet)
	if c.readInput+packet.Length < c.BufferLength {
		// 將 packet 當中的數據複製到 readBuffer 當中
		copy(c.readBuffer[c.readInput:c.readInput+packet.Length], packet.Data[:packet.Length])
		// 更新下次塞值的起始位置
		c.readInput += packet.Length
	} else {
		// 若剩餘長度不足一個 MTU，則分成兩次讀取
		idx := c.BufferLength - c.readInput
		// 將數據寫到 readBuffer 的尾部(數據長度為 idx)
		copy(c.readBuffer[c.readInput:], packet.Data[:idx])
		// 更新下次塞值的起始位置
		c.readInput = packet.Length - idx
		// 回到 readBuffer 最前面，將剩下的數據寫完(數據長度為 packet.Length - idx)
		copy(c.readBuffer[:c.readInput], packet.Data[idx:])
	}
	// 更新可讀數據長度
	c.ReadableLength += packet.Length
}

// 從 readBuffer 讀取指定長度的數據
func (c *Conn) Read(data []byte, length int32) {
	if c.readOutput+length < c.BufferLength {
		copy(data[:length], c.readBuffer[c.readOutput:c.readOutput+length])
		c.readOutput += length
	} else {
		idx := c.BufferLength - c.readOutput
		// 讀到 readBuffer 的結尾(長度為 idx)
		copy(data[:idx], c.readBuffer[c.readOutput:])
		// 將剩餘指定長度讀完(長度為 length-idx)
		c.readOutput = length - idx
		copy(data[idx:length], c.readBuffer[:c.readOutput])
	}
	// 更新可讀數據長度
	c.ReadableLength -= length
}

// 根據 checker 函式，檢查是否已讀取到所需的數據(條件可能是 長度 或 換行符 等)
func (c *Conn) CheckReadable(checker func(buffer []byte, i int32, o int32, length int32) bool) bool {
	return checker(c.readBuffer, c.readInput, c.readOutput, c.ReadableLength)
}

func (c *Conn) Write(data []byte, length int32) {
	packet := GetPacket()
	copy(packet.Data[:length], data[:length])
	packet.Length = length
	c.WriteCh <- packet
}

func (c *Conn) writeHandler(ctx context.Context) {
	defer c.wg.Done()
	defer func() {
		// 關閉寫入通道，表示不再發送數據
		if err := c.NetConn.(*net.TCPConn).CloseWrite(); err != nil {
			fmt.Println("Error closing write:", err.Error())
			return
		}
		if r := recover(); r != nil {
			fmt.Printf("recover: %+v\n", r)
		}
	}()
	var packet *Packet
	for {
		select {
		case <-ctx.Done():
			return
		case packet = <-c.WriteCh:
			c.write(packet)
		}
	}
}

func (c *Conn) write(packet *Packet) error {
	defer PutPacket(packet)
	var nWrite int
	var nWrite32 int32
	var err error
	for packet.Length > 0 {
		// 將數據寫出(Write 為阻塞型函式)
		nWrite, err = c.NetConn.Write(packet.Data[packet.Index : packet.Index+packet.Length])
		if err != nil {
			log.Error("Error writing, err: %+v\n", err)
			return errors.Wrapf(err, "Failed to write packet for conn-%d", c.GetId())
		}
		nWrite32 = int32(nWrite)
		packet.Index += nWrite32
		packet.Length -= nWrite32
	}
	return nil
}

// 當有需要重新連線的情況下，首先就會發生 Socket 讀取異常，並導致 Handler 的 goroutine 結束，因此無須再利用 c.stopCh 將 Handler 結束
func (c *Conn) Reconnect() {
	log.Info("cid: %d", c.id)
	if c.NetConn != nil {
		// 關閉當前連線
		c.NetConn.Close()
		// 清空連線物件
		c.NetConn = nil
	}
	c.writeErr = nil
}

func (c *Conn) Stop() {
	if c.cancel != nil {
		c.cancel()
	}
}

func (c *Conn) Release() {
	fmt.Println("Conn Release")
	// 關閉當前連線
	c.NetConn.Close()
	// 清空連線物件
	c.NetConn = nil
	// 狀態設置為未使用
	c.State = define.Unused
	c.writeErr = nil
	// 重置讀取用索引值
	c.readInput = 0
	c.readOutput = 0
	// 重置可讀取長度
	c.ReadableLength = 0
	// 重置可寫出長度
	c.WritableLength = 0
}

func (c *Conn) String() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("Conn(id: %d, ", c.id))
	b.WriteString(fmt.Sprintf("NetConn: %+v, State: %s, ", c.NetConn, c.State))
	b.WriteString(fmt.Sprintf("readInput: %d, readOutput: %d, ReadableLength: %d", c.readInput, c.readOutput, c.ReadableLength))
	return b.String()
}
