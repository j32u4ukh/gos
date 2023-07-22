package gos

import (
	"fmt"
	"net"

	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"

	"github.com/pkg/errors"
)

type goserver struct {
	// key: port; value: *Anser
	anserMap map[int32]ans.IAnswer
	// key: server id; value: *Asker
	askerMap map[int32]ask.IAsker
	// 啟動後，最大的 id 值 + 1，作為動態建立 Asker 時的 id 值
	nextServerId int32
}

func newGoserver() *goserver {
	g := &goserver{
		anserMap:     map[int32]ans.IAnswer{},
		askerMap:     map[int32]ask.IAsker{},
		nextServerId: 0,
	}
	return g
}

// 指定要監聽的 port，並生成 Anser 物件
func (g *goserver) listen(socketType define.SocketType, port int32) (ans.IAnswer, error) {
	if _, ok := g.anserMap[port]; !ok {
		laddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
		anser, err := ans.NewAnser(socketType, laddr, 10, 10)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create an Anser for port %d.", port)
		}

		g.anserMap[port] = anser
	}

	return g.anserMap[port], nil
}

// 向位置 ip:port 送出連線請求，利用 serverId 來識別多個連線
// serverId: server id
// ip: server ip
// port: server port
// socketType: 協定類型
func (g *goserver) bind(serverId int32, ip string, port int, socketType define.SocketType, onEvents base.OnEventsFunc, introduction *[]byte) (ask.IAsker, error) {
	if _, ok := g.askerMap[serverId]; !ok {
		laddr := &net.TCPAddr{IP: net.ParseIP(ip), Port: port, Zone: ""}
		asker, err := ask.NewAsker(socketType, serverId, laddr, 10, onEvents, introduction)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create an Asker for %s:%d.", ip, port)
		}

		g.askerMap[serverId] = asker
	}

	return g.askerMap[serverId], nil
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
