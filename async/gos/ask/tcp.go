package ask

import (
	"context"
	"encoding/binary"
	"fmt"
	"sync"
	"time"

	"github.com/j32u4ukh/gos/async/gos/base"
	"github.com/j32u4ukh/gos/async/gos/gtcp"
	"github.com/j32u4ukh/gos/utils/log"
	"github.com/pkg/errors"
)

type TcpAsker struct {
	*Asker
	order           binary.ByteOrder
	tcpPool         *sync.Pool
	workHandlerFunc func(tcp *gtcp.TcpContext) error
}

func NewTcpAsker(ip string, port int32, nConnect int32) *TcpAsker {
	asker := NewAsker(ip, int(port), nConnect)
	asker.readTimeout = 5000 * time.Millisecond
	a := &TcpAsker{
		Asker: asker,
		order: binary.LittleEndian,
	}
	// ===== 自定義函式 =====
	a.handlerFunc = a.Handler
	return a
}

func (a *TcpAsker) SetOrder(order binary.ByteOrder) {
	a.order = order
}

func (a *TcpAsker) Init() error {
	err := a.Asker.Init()
	if err != nil {
		return errors.Wrap(err, "Failed to initialize anser core server.")
	}
	a.tcpPool = &sync.Pool{
		New: func() any {
			return gtcp.NewTcpContext(a.contextBufferSize, a.order)
		},
	}
	return nil
}

// TODO: 檢查前導碼
func (a *TcpAsker) Handler(baseConn *base.Conn) {
	// 初始化 Tcp 和連線對象
	tcp := a.getTcp(baseConn)
	defer a.PutTcp(tcp)
	defer a.EndHandler(tcp.GetId())
	// 建立上下文，用於控制協程結束
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// 設置上下文
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

func (a *TcpAsker) process(tcp *gtcp.TcpContext) error {
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
			err := a.workHandler(tcp)
			if err != nil {
				return errors.Wrap(err, "Error occured while context processing")
			}
		default:
			return fmt.Errorf("invalid tcp state: %d", tcp.State)
		}
	}
	return nil
}

func (a *TcpAsker) Write(tcp *gtcp.TcpContext) error {
	data := tcp.Data.GetData()
	tcp.GetConn().Write(data, int32(len(data)))
	return nil
}

func (a *TcpAsker) SetWorkHandler(workHandlerFunc func(tcp *gtcp.TcpContext) error) {
	a.workHandlerFunc = workHandlerFunc
}

// 處理業務邏輯
func (a *TcpAsker) workHandler(tcp *gtcp.TcpContext) error {
	err := tcp.FormatData()
	if err != nil {
		return errors.Wrap(err, "Error occurred while data formating")
	}
	err = a.workHandlerFunc(tcp)
	if err != nil {
		return errors.Wrap(err, "Error occurred while work hanlding")
	}
	// 重置 欲讀取長度 以及 狀態值
	tcp.Reset()
	return nil
}

func (a *TcpAsker) getTcp(baseConn *base.Conn) *gtcp.TcpContext {
	tcp := a.tcpPool.Get().(*gtcp.TcpContext)
	tcp.SetConn(baseConn)
	a.contextMap[tcp.GetId()] = tcp
	return tcp
}

func (a *TcpAsker) GetTcp(id int32) (*gtcp.TcpContext, bool) {
	if ic, ok := a.contextMap[id]; ok {
		return ic.(*gtcp.TcpContext), true
	}
	return nil, false
}

func (a *TcpAsker) PutTcp(tcp *gtcp.TcpContext) {
	if tcp == nil {
		return
	}
	delete(a.contextMap, tcp.GetId())
	tcp.Reset()
	a.tcpPool.Put(tcp)
}
