package utils

import (
	"time"

	"github.com/j32u4ukh/gos/define"
)

// TODO: 只是提供各個伺服器建立時的預設數值，而非所有伺服器都必須共用此處設定
var GosConfig *Config

type Config struct {
	AnserReadTimeout     time.Duration
	AnswerReadBuffer     int32
	ConnBufferSize       int32
	DisconnectTime       time.Duration
	AnswerConnectNumbers map[define.SocketType]int32
	AnswerWorkNumbers    map[define.SocketType]int32
	AskerWorkNumbers     map[define.SocketType]int32
}

func init() {
	GosConfig = &Config{
		AnserReadTimeout: 5000 * time.Millisecond,
		AnswerReadBuffer: 64 * 1024,
		ConnBufferSize:   10,
		DisconnectTime:   time.Duration(3),
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
