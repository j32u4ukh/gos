package utils

import (
	"time"

	"github.com/j32u4ukh/gos/define"
)

var GosConfig *Config

type Config struct {
	HttpAnserReadTimeout time.Duration
	AnswerReadBuffer     int32
	ConnBufferSize       int32
	DisconnectTime       time.Duration
	AnswerConnectNumbers map[define.SocketType]int32
	AnswerWorkNumbers    map[define.SocketType]int32
	AskerWorkNumbers     map[define.SocketType]int32
}

func init() {
	GosConfig = &Config{
		HttpAnserReadTimeout: 5000 * time.Millisecond,
		AnswerReadBuffer:     64 * 1024,
		ConnBufferSize:       10,
		DisconnectTime:       time.Duration(3),
		AnswerConnectNumbers: map[define.SocketType]int32{
			define.Tcp0: 10,
			define.Http: 10,
		},
		AnswerWorkNumbers: map[define.SocketType]int32{
			define.Tcp0: 10,
			define.Http: 10,
		},
		AskerWorkNumbers: map[define.SocketType]int32{
			define.Tcp0: 10,
			define.Http: 10,
		},
	}
}
