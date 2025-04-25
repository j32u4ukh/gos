package gtcp

import (
	"encoding/binary"
	"fmt"

	"github.com/j32u4ukh/gos/base"
	"github.com/pkg/errors"
)

type TcpContext struct {
	*base.Context
	// 讀取狀態值 | 0: 讀取數據長度, 1: 根據前一階段取得的長度，讀取數據
	State       int8
	HeaderSize  int32
	ReadLength  int32
	isCompleted bool
}

func NewTcpContext(conn *base.Conn) *TcpContext {
	ctx := base.NewContext(conn)
	return &TcpContext{
		Context:     ctx,
		State:       0,
		HeaderSize:  4,
		ReadLength:  4,
		isCompleted: false,
	}
}

func (c *TcpContext) Read() error {
	var order binary.ByteOrder = c.Conn.GetBinaryOrder()
	var data []byte
	var err error
	// 檢查當前封包的數據是否滿足：可讀長度 大於 欲讀取長度
	for c.GetReadableLength() >= c.ReadLength {
		switch c.State {
		case 0:
			// 重置 TcpContext 狀態
			c.Release()
			// 從 readBuffer 當中讀取封包長度
			data, err = c.Conn.Read(uint32(c.HeaderSize))
			if err != nil {
				return errors.Wrap(err, "Error occured while context processing")
			}
			// 下次欲讀取長度為封包長度
			c.ReadLength = base.BytesToInt32(data, order)
			// 更新 TcpContext 狀態值
			c.State = 1
		case 1:
			// 讀取傳入的數據
			data, err = c.Conn.Read(uint32(c.ReadLength))
			if err != nil {
				return errors.Wrap(err, "Error occured while context processing")
			}
			c.AddRawData(data)
			c.ReadLength = c.HeaderSize
			c.isCompleted = true
			// 更新 TcpContext 狀態值
			c.State = 0
		default:
			return fmt.Errorf("invalid tcp state: %d", c.State)
		}
	}
	return nil
}

func (c *TcpContext) IsCompleted() bool {
	return c.isCompleted
}
