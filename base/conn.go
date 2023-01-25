package base

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/j32u4ukh/gos/define"

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

	// ==================================================
	// 連線結構
	// ==================================================
	// 連線編號
	Index int32
	// 連線物件
	NetConn net.Conn
	// 連線狀態
	State define.ConnectState
	// 連線模式(當 client 和 server 通信時對於長鏈接如何進行處理。)
	Mode ConnMode
	// 斷線時間戳(數秒後才切斷連線，預留時間給對方讀取數據)
	DisconnectTime time.Time
	// 下一個連線結構的指標
	Next *Conn
	// Handler 中斷用 chan
	stopCh chan bool

	// ==================================================
	// 讀寫結構
	// ==================================================
	// 緩衝長度
	BufferLength int32
	// 封包個數
	nPacket int32
	// 位元組順序 (Byte Order)，即 位元組 的排列順序
	order binary.ByteOrder
	// ========== 讀取 ==========
	// 讀取緩衝
	readBuffer []byte
	// 讀取輸入索引值(從這個位置開始往後寫入)
	readInput int32
	// 讀取輸出索引值(從這個位置開始往後取出數據，最多到 readInput)
	readOutput int32
	// 可讀取長度(readInput ~ readOutput 之間的數據量)
	ReadableLength int32
	// 下次讀取長度
	ReadLength int32
	// 讀取封包
	readPackets []*Packet
	// 讀取封包通道
	ReadCh chan *Packet
	// 讀取封包索引值
	readIdx int32
	// 封包長度
	// PacketLength int32
	// ========== 寫出 ==========
	// 寫出緩衝
	writeBuffer []byte
	// 寫出起始索引值(從這個位置開始往後取出要寫出得數據)
	writeInput int32
	// 此次寫出長度
	writeOutput int32
	// 可寫出長度(writeInput ~ writeOutput 之間的數據量)
	WritableLength int32

	// ==================================================
	// 暫存變數(避免重複宣告變數)
	// ==================================================
	nRead    int
	readErr  error
	nWrite   int
	writeIdx int32
	writeErr error
}

func NewConn(id int32, size int32) *Conn {
	c := &Conn{
		id:             id,
		Index:          -1,
		NetConn:        nil,
		State:          define.Unused,
		Next:           nil,
		stopCh:         make(chan bool, 1),
		BufferLength:   size * define.MTU,
		nPacket:        size,
		order:          binary.LittleEndian,
		readBuffer:     nil,
		readInput:      0,
		readOutput:     0,
		ReadableLength: 0,
		ReadLength:     define.DATALENGTH,
		readPackets:    []*Packet{},
		ReadCh:         make(chan *Packet, size),
		readIdx:        0,
		// PacketLength:   -1,
		writeBuffer:    nil,
		writeInput:     0,
		writeOutput:    0,
		writeIdx:       0,
		WritableLength: 0,
	}

	c.readBuffer = make([]byte, c.BufferLength)
	c.writeBuffer = make([]byte, c.BufferLength)

	var i int32
	for i = 0; i < size; i++ {
		c.readPackets = append(c.readPackets, NewPacket())
	}

	return c
}

// 取得連線物件編號
func (c *Conn) GetId() int32 {
	return c.id
}

func (c *Conn) Add(conn *Conn) {
	curr := c
	for curr.Next != nil {
		curr = curr.Next
	}
	curr.Next = conn
}

func (c *Conn) Handler() {
	// fmt.Printf("(c *Conn) handler, c.readErr: %+v\n", c.readErr)

	for c.readErr == nil {
		select {
		case <-c.stopCh:
			fmt.Printf("(c *Conn) handler | <-c.stopCh\n")
			return

		default:
			// fmt.Printf("(c *Conn) handler | readIdx: %d, netConn: %v\n", c.readIdx, c.NetConn != nil)

			// 每次讀取至多長度為 MTU 的數據(Read 為阻塞型函式)
			c.nRead, c.readErr = c.NetConn.Read(c.readPackets[c.readIdx].Data)
			// fmt.Printf("(c *Conn) handler | readIdx: %d, nRead: %d\n", c.readIdx, c.nRead)

			if c.readErr != nil {
				c.readPackets[c.readIdx].Error = c.readErr
				c.readPackets[c.readIdx].Length = 0
				// fmt.Printf("(c *Conn) handler | Read Error: %+v\n", c.readErr)

			} else {
				c.readPackets[c.readIdx].Error = nil
				c.readPackets[c.readIdx].Length = int32(c.nRead)
			}

			// 將讀取到的封包加入通道
			c.ReadCh <- c.readPackets[c.readIdx]
			c.readIdx += 1

			if c.readIdx >= c.nPacket {
				c.readIdx = 0
			}
		}
	}
}

// 讀取封包數據，並寫入 readBuffer
func (c *Conn) SetReadBuffer(packet *Packet) {
	// fmt.Printf("(c *Conn) setReadBuffer | before readOutput: %d, readInput: %d, readableLength: %d\n", c.readOutput, c.readInput, c.ReadableLength)

	// 更新可讀數據長度
	c.ReadableLength += packet.Length

	if c.readInput+packet.Length < c.BufferLength {
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

	// fmt.Printf("(c *Conn) setReadBuffer | after readOutput: %d, readInput: %d, readableLength: %d\n", c.readOutput, c.readInput, c.ReadableLength)
}

// 從 readBuffer 讀取指定長度的數據
func (c *Conn) Read(data *[]byte, length int32) {
	// fmt.Printf("(c *Conn) read | before readOutput: %d, readInput: %d, ReadableLength: %d, length: %d\n", c.readOutput, c.readInput, c.ReadableLength, length)

	// 更新可讀數據長度
	c.ReadableLength -= length

	if c.readOutput+length < c.BufferLength {
		copy((*data)[:length], c.readBuffer[c.readOutput:c.readOutput+length])
		c.readOutput += length

	} else {
		idx := c.BufferLength - c.readOutput

		// 讀到 readBuffer 的結尾(長度為 idx)
		copy((*data)[:idx], c.readBuffer[c.readOutput:])

		// 將剩餘指定長度讀完(長度為 length-idx)
		c.readOutput = length - idx
		copy((*data)[idx:length], c.readBuffer[:c.readOutput])
	}

	// fmt.Printf("(c *Conn) read | after readOutput: %d, readInput: %d, ReadableLength: %d\n", c.readOutput, c.readInput, c.ReadableLength)
}

// 根據 checker 函式，檢查是否已讀取到所需的數據(條件可能是 長度 或 換行符 等)
func (c *Conn) CheckReadable(checker func(buffer *[]byte, i int32, o int32, length int32) bool) bool {
	return checker(&c.readBuffer, c.readInput, c.readOutput, c.ReadableLength)
}

// 將寫出數據加入緩存
// TODO: 檢查 c.writeInput 是否反超 c.writeOutput，若反超，表示緩衝大小不足
func (c *Conn) SetWriteBuffer(data *[]byte, length int32) {
	// fmt.Printf("(c *Conn) setWriteBuffer | c.writeInput: %d, length: %d\n", c.writeInput, length)
	c.WritableLength += length

	if c.writeInput+length < c.BufferLength {
		copy(c.writeBuffer[c.writeInput:c.writeInput+length], (*data)[:length])
		c.writeInput += length

	} else {
		c.writeIdx = c.BufferLength - c.writeInput
		copy(c.writeBuffer[c.writeInput:], (*data)[:c.writeIdx])

		c.writeInput = length - c.writeIdx
		copy(c.writeBuffer[:c.writeInput], (*data)[c.writeIdx:length])
	}

	// fmt.Printf("(c *Conn) setWriteBuffer | c.writeInput: %d\n", c.writeInput)
}

func (c *Conn) Write() error {
	// fmt.Printf("(c *Conn) write | netConn: %v, writeInput: %d, writeOutput: %d\n", c.NetConn != nil, c.writeInput, c.writeOutput)

	for c.NetConn != nil && c.writeInput != c.writeOutput {

		if c.writeOutput < c.writeInput {
			// 將數據寫出(Write 為阻塞型函式)
			c.nWrite, c.writeErr = c.NetConn.Write(c.writeBuffer[c.writeOutput:c.writeInput])

			if c.writeErr != nil {
				fmt.Printf("(c *Conn) write | Failed to write data to conn(%d)\nwriteErr: %+v\n", c.Index, c.writeErr)
				return errors.Wrapf(c.writeErr, "Failed to write data to conn(%d)", c.Index)
			}

		} else {
			// 將封包數據寫出(Write 為阻塞型函式)
			c.nWrite, c.writeErr = c.NetConn.Write(c.writeBuffer[c.writeOutput:])

			if c.writeErr != nil {
				fmt.Printf("(c *Conn) write | Failed to write data to conn(%d)\nwriteErr: %+v\n", c.Index, c.writeErr)
				return errors.Wrapf(c.writeErr, "Failed to write data to conn(%d)", c.Index)
			}

		}

		// fmt.Printf("(c *Conn) write | Output: %+v\n", c.writeBuffer[c.writeOutput:c.writeOutput+int32(c.nWrite)])
		c.writeOutput += int32(c.nWrite)
		c.WritableLength -= int32(c.nWrite)

		if c.writeOutput == c.BufferLength {
			c.writeOutput = 0
		}
	}

	return nil
}

// 當有需要重新連線的情況下，首先就會發生 Socket 讀取異常，並導致 Handler 的 goroutine 結束，因此無須再利用 c.stopCh 將 Handler 結束
func (c *Conn) Reconnect() {
	fmt.Printf("(c *Conn) Reconnect | cid: %d\n", c.id)

	// 關閉當前連線
	c.NetConn.Close()

	// 清空連線物件
	c.NetConn = nil

	c.nRead = 0
	c.readErr = nil
	c.writeErr = nil
}

func (c *Conn) SetDisconnectTime(second time.Duration) {
	c.DisconnectTime = time.Now().Add(time.Second * second)
}

func (c *Conn) Release() {
	// 重置 Index
	c.Index = -1

	// 停止原本的 goroutine
	c.stopCh <- true

	// 關閉當前連線
	c.NetConn.Close()

	// 清空連線物件
	c.NetConn = nil

	// 狀態設置為未使用
	c.State = define.Unused

	// 釋放子節點
	c.Next = nil

	// 重置讀取用索引值
	c.readInput = 0
	c.readOutput = 0
}

func (c *Conn) String() string {
	var b bytes.Buffer
	b.WriteString(fmt.Sprintf("Conn(id: %d, Index: %d, ", c.id, c.Index))
	b.WriteString(fmt.Sprintf("NetConn: %+v, State: %s, Next: %+v, ", c.NetConn, c.State, c.Next))
	b.WriteString(fmt.Sprintf("readInput: %d, readOutput: %d, ReadableLength: %d", c.readInput, c.readOutput, c.ReadableLength))
	b.WriteString(fmt.Sprintf("writeInput: %d, writeOutput: %d)", c.writeInput, c.writeOutput))
	return b.String()
}

func CheckConns(root *Conn) {
	c := root
	for c != nil {
		fmt.Printf("CheckConns | %s\n", c)
		c = c.Next
	}
	fmt.Println()
}
