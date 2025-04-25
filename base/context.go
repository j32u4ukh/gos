package base

import "github.com/j32u4ukh/gos/cntr"

type Context struct {
	*Conn
	*cntr.BinaryData
}

func NewContext(conn *Conn) *Context {
	return &Context{
		Conn:       conn,
		BinaryData: cntr.NewBinaryData(),
	}
}

func (c *Context) Release() {
	c.BinaryData.Reset()
}
