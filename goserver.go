package gos

import (
	"fmt"
	"gos/ans"
	"gos/ask"
	"gos/base"
	"gos/define"
	"net"

	"github.com/pkg/errors"
)

type goserver struct {
	// key: port; value: *Anser
	anserMap map[int32]ans.IAnswer
	// key: site; value: *Asker
	askerMap map[int32]*ask.Asker
}

func newGoserver() *goserver {
	g := &goserver{
		anserMap: map[int32]ans.IAnswer{},
		askerMap: map[int32]*ask.Asker{},
	}
	return g
}

// 指定要監聽的 port，並生成 Anser 物件
func (g *goserver) listen(socketType define.SocketType, port int32) (ans.IAnswer, error) {
	if _, ok := g.anserMap[port]; !ok {
		laddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
		anser, err := ans.NewAnser(socketType, laddr, 10000, 10)

		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create an Anser for port %d.", port)
		}

		g.anserMap[port] = anser
	}

	return g.anserMap[port], nil
}

func (g *goserver) bind(site int32, ip string, port int, socketType define.SocketType) (*ask.Asker, error) {
	if _, ok := g.askerMap[site]; !ok {
		laddr := &net.TCPAddr{IP: net.ParseIP(ip), Port: port, Zone: ""}
		asker, err := ask.NewAsker(site, laddr, socketType, 10)

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
		fmt.Printf("CheckWorks | %s %s\n", msg, work)
		work = work.Next
	}
	fmt.Println()
}
