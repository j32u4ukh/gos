package utils

import (
	"time"

	"github.com/j32u4ukh/gos/define"
)

var GosConfig *Config

func init() {
	GosConfig = &Config{
		HttpAnserReadTimeout: 5000 * time.Millisecond,
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

type Config struct {
	HttpAnserReadTimeout time.Duration
	AnswerConnectNumbers map[define.SocketType]int32
	AnswerWorkNumbers    map[define.SocketType]int32
	AskerWorkNumbers     map[define.SocketType]int32
}
