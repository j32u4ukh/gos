package core

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/log"
	"github.com/j32u4ukh/gos/utils"
)

type Core struct {
	// 連線位置
	addr *net.TCPAddr
	// 讀取超時
	readTimeout time.Duration
	order       binary.ByteOrder
	// ==================================================
	// 連線列表
	// ==================================================
	connMap  map[int32]*base.Conn
	connPool *sync.Pool
	connId   int32
	// 當前連線數
	nConn int32
	// 最大連線數
	MAX_CONN       int32
	connBufferSize int32
	// ==================================================
	// Asker 數據包
	// ==================================================
	// 自我介紹數據
	introductionData []byte
	// 心跳包數據
	heartbeatData []byte
	// 心跳包間隔時間
	heartbeatLifetime time.Duration
	// 心跳包數據長度
	heartbeatLength int32
	// ==================================================
	// 注入函式
	// ==================================================
	handlerFunc func(baseConn *base.Conn) error
}

func NewCore(nConnect int32) *Core {
	c := &Core{
		readTimeout:       5 * time.Second,
		connMap:           make(map[int32]*base.Conn),
		connId:            -1,
		nConn:             0,
		MAX_CONN:          nConnect,
		connBufferSize:    10,
		heartbeatLifetime: 1 * time.Second,
	}
	c.connPool = &sync.Pool{
		New: func() any {
			c.UpdateNextConnId()
			baseConn := base.NewConn(atomic.LoadInt32(&c.connId), c.connBufferSize)
			baseConn.SetReadTimeout(c.readTimeout)
			return baseConn
		},
	}
	return c
}

func (c *Core) UpdateNextConnId() {
	// 使用 atomic 加以保證原子性
	id := atomic.AddInt32(&c.connId, 1)
	if id >= c.MAX_CONN {
		// 如果超過最大值，重置為 0
		atomic.StoreInt32(&c.connId, 0)
	}
}

func (c *Core) SetConnBufferSize(bufferSize int32) {
	c.connBufferSize = bufferSize
}

func (c *Core) SetReadTimeout(readTimeout time.Duration) {
	c.readTimeout = readTimeout
}

func (c *Core) SetOrder(order binary.ByteOrder) {
	c.order = order
}

func (c *Core) GetConn(cid int32, netConn net.Conn) *base.Conn {
	var baseConn *base.Conn
	var ok bool
	if baseConn, ok = c.connMap[cid]; !ok {
		baseConn = c.connPool.Get().(*base.Conn)
		cid = baseConn.GetId()
		c.connMap[cid] = baseConn
	}
	if netConn != nil {
		baseConn.SetNetConn(netConn)
	}
	return baseConn
}

func (c *Core) PutConn(baseConn *base.Conn) {
	if baseConn == nil {
		return
	}
	delete(c.connMap, baseConn.GetId())
	baseConn.Release()
	c.connPool.Put(baseConn)
}

func (c *Core) Handler(baseConn *base.Conn) {
	cid := baseConn.GetId()
	log.Debug("cid: %d", cid)
	c.connMap[cid] = baseConn
	defer func() {
		atomic.AddInt32(&c.nConn, -1)
		if r := recover(); r != nil {
			log.Error("Error occurred while handling conn-%d, err: %+v", cid, r)
		}
	}()
	ctx, cancel := context.WithCancel(context.Background())
	baseConn.SetCancelFunc(cancel)
	defer cancel()
	// 啟動連線物件處理
	go baseConn.Handler(ctx)
	var (
		errorCh *base.ConnError
		err     error
		timer   *time.Timer
	)
	// 僅在需要心跳時啟動 timer
	if c.heartbeatData != nil {
		timer = time.NewTimer(c.heartbeatLifetime)
		defer timer.Stop()
	}
	for {
		select {
		// 上下文結束
		case <-ctx.Done():
			return
		// 使用一個 helper 函式避免 nil panic
		case <-utils.GetSafetyTimerChan(timer):
			baseConn.Write(c.heartbeatData)
			c.resetHeartbeatTimer(timer)
		// 收到錯誤信號
		case errorCh = <-baseConn.ErrorCh():
			// TODO: 區分錯誤類型
			// TODO: 伺服端 | 區分是網路不穩而斷線，還是主動斷線
			// TODO: 客戶端 | 若是網路不穩而斷線而非主動斷線，則確認是否需要啟用重連線機制
			switch errorCh.Type {
			case base.ErrTypeNetwork:
			default:
				log.Info("errorCh: %+v", errorCh)
			}
			return
		// 收到讀取信號
		case <-baseConn.ReadCh():
			if baseConn.GetReadableLength() == 0 {
				return
			}
			// 讀取並處理封包
			if err = c.handlerFunc(baseConn); err != nil {
				log.Error("Error occurred while processing, err: %+v", err)
			}
			c.resetHeartbeatTimer(timer)
		}
	}
}

func (c *Core) SetHandlerFunc(handlerFunc func(baseConn *base.Conn) error) {
	c.handlerFunc = handlerFunc
}

func (c *Core) resetHeartbeatTimer(timer *time.Timer) {
	if timer == nil {
		return
	}
	// 確保計時器通道已經清空
	if !timer.Stop() {
		select {
		case <-timer.C: // 等待計時器的 channel 被清空
		default: // 如果已經沒有新事件，則不進行等待
		}
	}
	// 重置計時器
	timer.Reset(c.heartbeatLifetime)
}

func (c *Core) Disconnect(cid int32, duration time.Duration) error {
	if conn, ok := c.connMap[cid]; ok {
		if duration > 0 {
			time.Sleep(duration)
		}
		conn.Stop()
		return nil
	}
	return fmt.Errorf("不存在 cid 為 %d 的連線", cid)
}
