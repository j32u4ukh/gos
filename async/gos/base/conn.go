package base

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"

	"github.com/pkg/errors"
)

type ConnMode byte

const (
	CLOSE     ConnMode = 0
	KEEPALIVE ConnMode = 1
)

type ConnResult struct {
	N   int32
	Err error
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
	// 連線模式(當 client 和 server 通信時對於長鏈接如何進行處理。)
	Mode ConnMode
	// 斷線時間戳(數秒後才切斷連線，預留時間給對方讀取數據)
	DisconnectTime time.Time
	// Handler 中斷用 chan
	stopCh chan bool

	// ==================================================
	// 讀寫結構
	// ==================================================
	// 緩衝長度
	DataLength int32
	// 封包個數
	nPacket int32
	// 位元組順序 (Byte Order)，即 位元組 的排列順序
	order binary.ByteOrder
	// ========== 讀取 ==========
	// 讀取緩衝
	readData []byte
	// 讀取輸入索引值(從這個位置開始往後寫入)
	readInput int32
	// 讀取輸出索引值(從這個位置開始往後取出數據，最多到 readInput)
	readOutput int32
	// 可讀取長度(readInput ~ readOutput 之間的數據量)
	ReadableLength int32
	// 讀取封包通道
	ReadCh     chan *Packet
	readBuffer []byte
	// ========== 寫出 ==========
	WriteCh chan *Packet
	// 寫出緩衝
	writeBuffer []byte
	// 寫出起始索引值(從這個位置開始往後取出要寫出得數據)
	writeInput int32
	// 此次寫出長度
	writeOutput int32
	// 可寫出長度(writeInput ~ writeOutput 之間的數據量)
	WritableLength int32
	// ========== 其他 ==========
	ResultCh chan *Packet

	// ==================================================
	// 暫存變數(避免重複宣告變數)
	// ==================================================
	nWrite   int
	writeIdx int32
	writeErr error
}

func NewConn(id int32, size int32) *Conn {
	c := &Conn{
		id:             id,
		NetConn:        nil,
		State:          define.Unused,
		Mode:           KEEPALIVE,
		stopCh:         make(chan bool, 1),
		DataLength:     size * define.MTU,
		nPacket:        size,
		order:          binary.LittleEndian,
		readData:       nil,
		readInput:      0,
		readOutput:     0,
		ReadableLength: 0,
		ReadCh:         make(chan *Packet, size),
		readBuffer:     make([]byte, MTU),
		WriteCh:        make(chan *Packet, size),
		writeBuffer:    nil,
		writeInput:     0,
		writeOutput:    0,
		writeIdx:       0,
		WritableLength: 0,
		ResultCh:       make(chan *Packet, size),
	}

	c.readData = make([]byte, c.DataLength)
	c.writeBuffer = make([]byte, c.DataLength)

	return c
}

// 取得連線物件編號
func (c *Conn) GetId() int32 {
	return c.id
}

func (c *Conn) Handler(ctx context.Context) {
	handlerCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.wg.Add(2)
	go c.readHandler(handlerCtx)
	go c.writeHandler(handlerCtx)
	c.release()
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
	var nRead32 int32
	for {
		select {
		case <-ctx.Done():
			return
		default:
			packet = GetPacket()
			// 每次讀取至多長度為 MTU 的數據(Read 為阻塞型函式)
			nRead, packet.Error = c.NetConn.Read(c.readBuffer)
			// utils.Debug("readIdx: %d, nRead: %d", c.readIdx, c.nRead)
			packet.Length = int32(nRead)

			if packet.Error == nil {

				// 更新可讀數據長度
				c.ReadableLength += nRead32

				if c.readInput+nRead32 < c.DataLength {
					copy(c.readData[c.readInput:c.readInput+nRead32], c.readBuffer[:nRead32])

					// 更新下次塞值的起始位置
					c.readInput += nRead32

				} else {
					// 若剩餘長度不足一個 MTU，則分成兩次讀取
					idx := c.DataLength - c.readInput

					// 將數據寫到 readBuffer 的尾部(數據長度為 idx)
					copy(c.readData[c.readInput:], c.readBuffer[:idx])

					// 更新下次塞值的起始位置
					c.readInput = nRead32 - idx

					// 回到 readBuffer 最前面，將剩下的數據寫完(數據長度為 packet.Length - idx)
					copy(c.readData[:c.readInput], c.readBuffer[idx:])
				}
			}

			// 將讀取到的封包加入通道
			c.ReadCh <- packet
		}
	}
}

func (c *Conn) writeHandler(ctx context.Context) {
	defer func() {
		// 關閉寫入通道，表示不再發送數據
		if err := c.NetConn.(*net.TCPConn).CloseWrite(); err != nil {
			fmt.Println("Error closing write:", err.Error())
			return
		}
		if r := recover(); r != nil {
			fmt.Printf("recover: %+v\n", r)
		}
		c.wg.Done()
	}()
	var packet, resultRacket *Packet
	var nWrite int
	var nWrite32 int32
	var err error
	for {
		select {
		case <-ctx.Done():
			return
		case packet = <-c.WriteCh:
			for packet.Length > 0 {
				// 將數據寫出(Write 為阻塞型函式)
				nWrite, err = c.NetConn.Write(packet.Data[packet.Index : packet.Index+packet.Length])
				if err != nil {
					fmt.Printf("Error writing, err: %+v", err)
					resultRacket = GetPacket()
					c.ResultCh <- resultRacket
					return
				}
				nWrite32 = int32(nWrite)
				packet.Index += nWrite32
				packet.Length -= nWrite32
			}
			PutPacket(packet)
		}
	}
}

// 從 readBuffer 讀取指定長度的數據
func (c *Conn) Read(data *[]byte, length int32) {
	// 更新可讀數據長度
	c.ReadableLength -= length

	if c.readOutput+length < c.DataLength {
		copy((*data)[:length], c.readData[c.readOutput:c.readOutput+length])
		c.readOutput += length

	} else {
		idx := c.DataLength - c.readOutput

		// 讀到 readBuffer 的結尾(長度為 idx)
		copy((*data)[:idx], c.readData[c.readOutput:])

		// 將剩餘指定長度讀完(長度為 length-idx)
		c.readOutput = length - idx
		copy((*data)[idx:length], c.readData[:c.readOutput])
	}
}

// 根據 checker 函式，檢查是否已讀取到所需的數據(條件可能是 長度 或 換行符 等)
func (c *Conn) CheckReadable(checker func(buffer *[]byte, i int32, o int32, length int32) bool) bool {
	return checker(&c.readData, c.readInput, c.readOutput, c.ReadableLength)
}

// 將寫出數據加入緩存
// TODO: 檢查 c.writeInput 是否反超 c.writeOutput，若反超，表示緩衝大小不足
func (c *Conn) SetWriteBuffer(data *[]byte, length int32) {
	c.WritableLength += length
	if c.writeInput+length < c.DataLength {
		copy(c.writeBuffer[c.writeInput:c.writeInput+length], (*data)[:length])
		c.writeInput += length
	} else {
		c.writeIdx = c.DataLength - c.writeInput
		copy(c.writeBuffer[c.writeInput:], (*data)[:c.writeIdx])
		c.writeInput = length - c.writeIdx
		copy(c.writeBuffer[:c.writeInput], (*data)[c.writeIdx:length])
	}
}

func (c *Conn) Write() error {
	for c.NetConn != nil && c.writeInput != c.writeOutput {

		if c.writeOutput < c.writeInput {
			// 將數據寫出(Write 為阻塞型函式)
			c.nWrite, c.writeErr = c.NetConn.Write(c.writeBuffer[c.writeOutput:c.writeInput])

			if c.writeErr != nil {
				utils.Error("Failed to write data to conn(%d), writeErr: %+v", c.id, c.writeErr)
				return errors.Wrapf(c.writeErr, "Failed to write data to conn(%d)", c.id)
			}

		} else {
			// 將封包數據寫出(Write 為阻塞型函式)
			c.nWrite, c.writeErr = c.NetConn.Write(c.writeBuffer[c.writeOutput:])

			if c.writeErr != nil {
				utils.Error("Failed to write data to conn(%d), writeErr: %+v", c.id, c.writeErr)
				return errors.Wrapf(c.writeErr, "Failed to write data to conn(%d)", c.id)
			}

		}

		c.writeOutput += int32(c.nWrite)
		c.WritableLength -= int32(c.nWrite)

		if c.writeOutput == c.DataLength {
			c.writeOutput = 0
		}
	}

	return nil
}

// 當有需要重新連線的情況下，首先就會發生 Socket 讀取異常，並導致 Handler 的 goroutine 結束，因此無須再利用 c.stopCh 將 Handler 結束
func (c *Conn) Reconnect() {
	utils.Info("cid: %d", c.id)

	if c.NetConn != nil {
		// 關閉當前連線
		c.NetConn.Close()

		// 清空連線物件
		c.NetConn = nil
	}

	c.writeErr = nil
}

func (c *Conn) SetDisconnectTime(second time.Duration) {
	c.DisconnectTime = time.Now().Add(time.Second * second)
}

func (c *Conn) Release() {
	fmt.Println("Conn Release")
	// 停止原本的 goroutine
	c.stopCh <- true
}

func (c *Conn) release() {
	fmt.Println("Conn release")

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

	// 重置寫出用索引值
	c.writeInput = 0
	c.writeOutput = 0

	// 重置可寫出長度
	c.WritableLength = 0
	c.writeIdx = 0
}

func (c *Conn) String() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("Conn(id: %d, ", c.id))
	b.WriteString(fmt.Sprintf("NetConn: %+v, State: %s, ", c.NetConn, c.State))
	b.WriteString(fmt.Sprintf("readInput: %d, readOutput: %d, ReadableLength: %d", c.readInput, c.readOutput, c.ReadableLength))
	b.WriteString(fmt.Sprintf("writeInput: %d, writeOutput: %d)", c.writeInput, c.writeOutput))
	return b.String()
}
