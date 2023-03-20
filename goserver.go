package gos

import (
	"fmt"
	"net"

	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/define"

	"github.com/pkg/errors"
)

type goserver struct {
	// key: port; value: *Anser
	anserMap map[int32]ans.IAnswer
	// key: site; value: *Asker
	askerMap map[int32]ask.IAsker
	// 啟動後，最大的 site 值 + 1，作為動態建立 Asker 時的 site 值
	nextSite int32
}

func newGoserver() *goserver {
	g := &goserver{
		anserMap: map[int32]ans.IAnswer{},
		askerMap: map[int32]ask.IAsker{},
		nextSite: 0,
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

func (g *goserver) bind(site int32, ip string, port int, socketType define.SocketType, onConnect func()) (ask.IAsker, error) {
	if _, ok := g.askerMap[site]; !ok {
		laddr := &net.TCPAddr{IP: net.ParseIP(ip), Port: port, Zone: ""}
		asker, err := ask.NewAsker(socketType, site, laddr, 10, onConnect)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create an Asker for %s:%d.", ip, port)
		}

		g.askerMap[site] = asker
	}

	return g.askerMap[site], nil
}

//

func CheckWorks(msg string, root *base.Work) {
	work := root
	for work != nil {
		// fmt.Printf("CheckWorks | %s %s\n", msg, work)
		logger.Debug("CheckWorks | %s %s", msg, work)
		work = work.Next
	}
	fmt.Println()
}
