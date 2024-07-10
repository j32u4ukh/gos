package base

import (
	"context"
	"net"
)

type Context struct {
	context.Context
	net.Conn
}

func NewContext(ctx context.Context, conn net.Conn) *Context {
	return &Context{
		Context: ctx, 
		Conn: conn,
	}
}