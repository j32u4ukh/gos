package core

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/log"

	"github.com/pkg/errors"
)

type IAnser interface {
	// 開始監聽
	Listen()
	// 執行
	Handler(baseConn *base.Conn)
	// 主動中斷指定連線
	Disconnect(cid int32, duration time.Duration) error
}

type Anser struct {
	*Core
	port int32
	// 監聽連線物件
	listener *net.TCPListener
}

func NewAnser(port int32, nConnect int32) *Anser {
	return &Anser{
		Core: NewCore(nConnect),
		port: port,
	}
}

func (a *Anser) Init() error {
	var err error
	a.addr, err = net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", a.port))
	if err != nil {
		return errors.Wrapf(err, "Failed to resolve tcp addr :%d", a.port)
	}
	log.Info("addr: %+v", a.addr)
	a.listener, err = net.ListenTCP("tcp", a.addr)
	if err != nil {
		return errors.Wrapf(err, "Failed to listen at port %d", a.port)
	}
	log.Info("listener: %+v", a.listener)
	return nil
}

// 監聽連線並註冊
func (a *Anser) Listen() {
	var netConn net.Conn
	var baseConn *base.Conn
	var err error
	for {
		netConn, err = a.listener.AcceptTCP()
		if err != nil {
			// log.Error("接受客戶端連接異常: %+v", err.Error())
			// return
			// 判斷是否因 Listener 關閉而中斷
			if opErr, ok := err.(*net.OpError); ok && opErr.Op == "accept" {
				log.Info("監聽已停止")
				return
			}
			log.Error("接受客戶端連接異常: %+v", err)
			continue
		}
		log.Info("客戶端連接來自: %s", netConn.RemoteAddr())
		if atomic.LoadInt32(&a.nConn) < a.MAX_CONN {
			atomic.AddInt32(&a.nConn, 1)
			log.Info("連線數: %d", atomic.LoadInt32(&a.nConn))
			baseConn = a.GetConn(-1, netConn)
			baseConn.SetState(define.Connected)
			go a.Handler(baseConn)
		}
	}
}
