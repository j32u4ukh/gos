package gos

import (
	"time"

	"github.com/j32u4ukh/gos/async/gos/ans"
)

type goserver struct {
	// key: port; value: *Anser
	anserMap map[int32]ans.IAnser
	// key: server id; value: *Asker
	// askerMap map[int32]ask.IAsker
	// 啟動後，最大的 id 值 + 1，作為動態建立 Asker 時的 id 值
	nextServerId int32
	// 每幀時長
	frameTime time.Duration
}

func newGoserver() *goserver {
	g := &goserver{
		anserMap: map[int32]ans.IAnser{},
		// askerMap:     map[int32]ask.IAsker{},
		nextServerId: 0,
		frameTime:    20 * time.Millisecond,
	}
	return g
}
