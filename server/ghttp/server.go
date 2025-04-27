package ghttp

import (
	"sync"

	"github.com/j32u4ukh/gos/base"
)

type Server struct {
	// ==================================================
	// Context
	// 個數與 Anser 的 nConnect 相同，因此可利用 Conn 中的 id 作為索引值，來存取,
	// 由於 Context 是使用 Conn 的 id 作為索引值，因此可以不用從第一個開始使用，結束使用後也不需要對順序進行調整
	// ==================================================
	contextPool sync.Pool
	contextMap  map[int32]*HttpContext
}

func NewServer(mode HttpMode) *Server {
	return &Server{
		contextPool: sync.Pool{New: func() any {
			context := NewHttpContext()
			context.setHttpMode(mode)
			return context
		}},
		contextMap: make(map[int32]*HttpContext),
	}
}

func (s *Server) GetContext(cid int32, baseConn *base.Conn) *HttpContext {
	var context *HttpContext
	var ok bool
	if context, ok = s.contextMap[cid]; !ok {
		context = s.contextPool.Get().(*HttpContext)
		s.contextMap[cid] = context
	}
	if baseConn != nil {
		context.Conn = baseConn
	}
	return context
}

func (s *Server) PutContext(context *HttpContext) {
	if context == nil {
		return
	}
	delete(s.contextMap, context.GetId())
	context.Release()
	s.contextPool.Put(context)
}
