package main

import (
	"fmt"
	"net"
)

func main() {
	laddr := &net.TCPAddr{IP: net.ParseIP("192.168.0.198"), Port: 1023, Zone: ""}
	netConn1, err := net.DialTCP("tcp", nil, laddr)

	if err != nil {
		fmt.Printf("err: %+v\n", err)
		return
	}

	netConn2, err := net.DialTCP("tcp", nil, laddr)

	if err != nil {
		fmt.Printf("err: %+v\n", err)
		return
	}

	fmt.Printf("netConn1: %+v\n", netConn1)
	fmt.Printf("netConn2: %+v\n", netConn2)

	netConn1.Write([]byte("netConn1"))
	netConn2.Write([]byte("netConn2"))
}
