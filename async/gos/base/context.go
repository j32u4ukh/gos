package base

import (
	"context"

	"github.com/j32u4ukh/cntr"
)

type IContext interface {
	GetId() int32
	GetCtx() context.Context
	GetConn() *Conn
	WithCancel() (IContext, context.CancelFunc)
}

type Context struct {
	ctx        context.Context
	conn       *Conn
	Buffer     []byte
	Data       *cntr.BinaryData
	bufferSize uint32
}

func NewContext(bufferSize uint32) *Context {
	return &Context{
		Buffer:     make([]byte, bufferSize),
		bufferSize: bufferSize,
	}
}

func (c *Context) GetId() int32 {
	if c.conn == nil {
		return -1
	}
	return c.conn.GetId()
}

func (c *Context) SetCtx(ctx context.Context) {
	c.ctx = ctx
}

func (c *Context) GetCtx() context.Context {
	return c.ctx
}

func (c *Context) SetConn(conn *Conn) {
	c.conn = conn
}

func (c *Context) GetConn() *Conn {
	return c.conn
}

func (c *Context) WithCancel() (IContext, context.CancelFunc) {
	ctx, cancel := context.WithCancel(c.ctx)
	newCtx := c.Clone()
	newCtx.ctx = ctx
	return newCtx, cancel
}

func (c *Context) Clone() *Context {
	clone := &Context{
		ctx:    c.ctx,
		conn:   c.conn,
		Data:   cntr.NewBinaryData(),
		Buffer: make([]byte, c.bufferSize),
	}
	raw := c.Data.GetData()
	clone.Data.AddRawData(raw)
	copy(clone.Buffer, c.Buffer)
	return clone
}

func (c *Context) Reset() {
	c.Data.Reset()
}

func (c *Context) Release() {
	c.ctx = nil
	c.conn = nil
	c.Data.Reset()
	c.Buffer = make([]byte, c.bufferSize)
}
