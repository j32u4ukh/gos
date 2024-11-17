package ans

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/async/gos/base"
	"github.com/j32u4ukh/gos/async/gos/gtcp"
	"github.com/j32u4ukh/gos/utils/log"

	"github.com/pkg/errors"
)

// ====================================================================================================
// TcpAnser
// ====================================================================================================

type TcpAnser struct {
	*Anser
	//
	order           binary.ByteOrder
	tcpPool         *sync.Pool
	tcp             *gtcp.TcpContext
	workHandlerFunc func(tcp *gtcp.TcpContext) error
}

func NewTcpAnser(port int32, nConnect int32) (*TcpAnser, error) {
	// ===== Anser =====
	anser, err := NewAnser(port, nConnect)
	if err != nil {
		return nil, errors.Wrapf(err, "Failed to new TcpAnser.")
	}
	anser.ReadTimeout = 5000 * time.Millisecond
	// ===== TcpAnser =====
	a := &TcpAnser{
		Anser: anser,
		order: binary.LittleEndian,
		tcpPool: &sync.Pool{
			New: func() any {
				return gtcp.NewTcp0()
			},
		},
		tcp: nil,
	}
	// ===== 自定義函式 =====
	a.handlerFunc = a.Handler
	return a, nil
}

func (a *TcpAnser) SetWorkHandler(workHandlerFunc func(tcp *gtcp.TcpContext) error) {
	a.workHandlerFunc = workHandlerFunc
}

// TODO: 檢查前導碼
func (a *TcpAnser) Handler(netConn net.Conn) {
	// 初始化 Tcp 和連線對象
	tcp := a.getTcp()
	baseConn := a.GetConn(netConn)
	defer a.PutTcp(tcp)
	defer a.EndHandler(tcp.GetId())
	// 建立上下文，用於控制協程結束
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 設置連線及上下文
	tcp.SetConn(baseConn)
	tcp.SetCtx(ctx)
	// 啟動連線的處理協程
	go baseConn.Handler(ctx)
	var err error
	for {
		select {
		// 上下文結束
		case <-ctx.Done():
			return
		// 收到讀取信號
		case <-tcp.GetConn().ReadCh:
			// 讀取並處理封包
			/*
							數據包解析錯誤表明客戶端發送的數據格式嚴重異常（如協議不符、惡意數據）。
			錯誤無法恢覆，繼續保持連接可能導致資源浪費或安全隱患。
			系統更傾向於快速清理問題連接而非糾錯。

			在實際項目中，可以根據以下條件選擇適合的策略：

			    低延遲優先：優先跳過當前數據包（策略 1）。
			    安全性優先：優先關閉連接（策略 2）。
			    容錯性優先：嘗試錯誤恢覆（策略 3）。
			    覆雜協議或動態需求：按錯誤類型動態處理（策略 4）。

			如果不確定，可以從 打印錯誤並跳過當前數據包 入手，逐步增加策略覆雜度。
			*/
			if err = a.process(tcp); err != nil {
				log.Error("Error occurred while processing, err: %+v", err)
				return
			}
		}
	}
}

func (a *TcpAnser) process(tcp *gtcp.TcpContext) error {
	baseConn := tcp.GetConn()
	for baseConn.CheckReadable(tcp.ReadableChecker) {
		switch tcp.State {
		case 0:
			// 從 readBuffer 當中讀取封包長度
			baseConn.Read(tcp.Buffer, tcp.HeaderSize)
			// 下次欲讀取長度為封包長度
			tcp.ReadLength = base.BytesToInt32(tcp.Buffer[:tcp.HeaderSize], a.order)
			// 更新 currTcp0 狀態值
			tcp.State = 1
		case 1:
			// 讀取傳入的數據
			baseConn.Read(tcp.Buffer, tcp.ReadLength)
			tcp.Data.AddRawData(tcp.Buffer[:tcp.ReadLength])
			// 處理業務邏輯
			err := a.WorkHandler(tcp)
			if err != nil {
				return errors.Wrap(err, "Error occured while context processing")
			}
		default:
			return fmt.Errorf("invalid tcp state: %d", tcp.State)
		}
	}
	return nil
}

// 處理業務邏輯
func (a *TcpAnser) WorkHandler(tcp *gtcp.TcpContext) error {
	err := a.workHandlerFunc(tcp)
	if err != nil {
		return errors.Wrap(err, "Error occurred while work hanlding")
	}
	// 重置 欲讀取長度 以及 狀態值
	tcp.Reset()
	return nil
}

func (a *TcpAnser) Write(tcp *gtcp.TcpContext) {
	// TODO: tcp to raw data
	data := []byte{}
	tcp.GetConn().Write(data, int32(len(data)))
}

func (a *TcpAnser) getTcp() *gtcp.TcpContext {
	tcp := a.tcpPool.Get().(*gtcp.TcpContext)
	a.contextMap[tcp.GetId()] = tcp
	return tcp
}

func (a *TcpAnser) PutTcp(tcp *gtcp.TcpContext) {
	if tcp == nil {
		return
	}
	delete(a.contextMap, tcp.GetId())
	tcp.Reset()
	a.tcpPool.Put(tcp)
}
