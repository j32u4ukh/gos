package gos

import (
	"fmt"
	"time"

	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/utils"
)

type goserver struct {
	// key: port; value: *Anser
	anserMap map[int32]ans.IAnswer
	// key: server id; value: *Asker
	askerMap map[int32]ask.IAsker
	// 啟動後，最大的 id 值 + 1，作為動態建立 Asker 時的 id 值
	nextServerId int32
	// 每幀時長
	frameTime time.Duration
}

func newGoserver() *goserver {
	g := &goserver{
		anserMap:     map[int32]ans.IAnswer{},
		askerMap:     map[int32]ask.IAsker{},
		nextServerId: 0,
		frameTime:    20 * time.Millisecond,
	}
	return g
}

//
func CheckWorks(msg string, root *base.Work) {
	work := root
	for work != nil {
		// fmt.Printf("CheckWorks | %s %s\n", msg, work)
		utils.Debug("CheckWorks | %s %s", msg, work)
		work = work.Next
	}
	fmt.Println()
}
