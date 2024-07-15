package base

import (
	"context"
)

type IContext interface {
	GetCtx() context.Context
	GetConn() *Conn
	WithCancel() (IContext, context.CancelFunc)
}

type Context struct {
	ctx  context.Context
	conn *Conn
}

func NewContext(ctx context.Context, conn *Conn) *Context {
	return &Context{
		ctx:  ctx,
		conn: conn,
	}
}

func (c *Context) GetCtx() context.Context {
	return c.ctx
}

func (c *Context) GetConn() *Conn {
	return c.conn
}

func (c *Context) WithCancel() (IContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	newCtx := NewContext(ctx, c.conn)
	return newCtx, cancel
}

func ContextWithCancel(c IContext) (IContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.GetCtx())
	newCtx := NewContext(ctx, c.GetConn())
	return newCtx, cancel
}
