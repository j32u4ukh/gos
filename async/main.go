package main

import (
	"bytes"
	"fmt"
	"net"
	"time"

	"github.com/j32u4ukh/gos/async/ans"
	"github.com/j32u4ukh/gos/define"
)

func main() {
	ServerDemo()
}

func ServerDemo() {
	port := 5000
	laddr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		PrintError("Error resolving tcp address, err: %+v", err)
		return
	}
	anser, err := ans.NewAnser(define.Tcp0, laddr, 10)
	if err != nil {
		PrintError("Error creating anser, err: %+v", err)
		return
	}
	go anser.Listen()
	for {
		select {
		default:
			fmt.Println("ServerDemo is running...")
			time.Sleep(1 * time.Second)
		}
	}
}

func PrintError(format string, v ...any) {
	format = fmt.Sprintf("[Error] %s\n", format)
	fmt.Printf(format, v...)
}

func handleRequest(conn net.Conn) {
	// Make a buffer to hold incoming data.
	buf := make([]byte, 4096)
	// Read the incoming connection into the buffer.
	l, err := conn.Read(buf)
	if err != nil {
		fmt.Println("Error reading:", err.Error())
	}
	fmt.Println(l)

	/*
		71 69 84 32 47 101 110 100 63 100 49 61 51 50 38 100 50 61 119 111 114 100 32 72 84 84 80 47 49 46 49 13 10 67 111 110 116 101 110
		116 45 84 121 112 101 58 32 97 112 112 108 105 99 97 116 105 111 110 47 106 115 111 110 13 10 85 115 101 114 45 65 103 101 110 116
		58 32 80 111 115 116 109 97 110 82 117 110 116 105 109 101 47 55 46 50 57 46 50 13 10 65 99 99 101 112 116 58 32 42 47 42 13 10 80
		111 115 116 109 97 110 45 84 111 107 101 110 58 32 54 52 98 54 48 52 53 98 45 99 51 51 50 45 52 97 55 101 45 97 48 55 53 45 50 54
		101 52 55 97 100 50 98 54 54 97 13 10 72 111 115 116 58 32 49 57 50 46 49 54 56 46 48 46 49 57 56 58 51 51 51 51 13 10 65 99 99 101
		112 116 45 69 110 99 111 100 105 110 103 58 32 103 122 105 112 44 32 100 101 102 108 97 116 101 44 32 98 114 13 10 67 111 110 110 101
		99 116 105 111 110 58 32 107 101 101 112 45 97 108 105 118 101 13 10 67 111 110 116 101 110 116 45 76 101 110 103 116 104 58 32 51
		53 13 10 13 10 123 13 10 32 32 32 32 34 105 100 34 58 48 44 13 10 32 32 32 32 34 109 115 103 34 58 34 116 101 115 116 34 13 10 125
	*/
	request := string(buf[:l])
	fmt.Println(SliceToString(buf[:l]))
	fmt.Println(request)

	fmt.Println("================================================================")
	// Accept: */*
	/*
		GET /end HTTP/1.1
		Content-Type: application/json
		User-Agent: PostmanRuntime/7.29.2
		Accept:
		Postman-Token: 6746eca0-5849-4c5f-a208-2d981c6100ff
		Host: 192.168.0.198:3333
		Accept-Encoding: gzip, deflate, br
		Connection: keep-alive
		Content-Length: 35

		{
			"id":0,
			"msg":"test"
		}


		POST /end HTTP/1.1
		Content-Type: application/json
		User-Agent: PostmanRuntime/7.29.2
		Accept:
		Postman-Token: 187963da-afd1-43a6-ae2b-430f67c50ffc
		Host: 192.168.0.198:3333
		Accept-Encoding: gzip, deflate, br
		Connection: keep-alive
		Content-Length: 35

		{
			"id":0,
			"msg":"test"
		}
	*/
	// Send a response back to person contacting us.
	r := []byte(fmt.Sprintf("HTTP/1.1 200 OK\r\nConnection: close\r\n%s: text/html\r\nContent-Length: 19\r\n\r\n<h1>Hola Mundo</h1>",
		"Content-Type"))
	conn.Write(r)
	// Close the connection when you're done with it.
	conn.Close()
}

func SliceToString[T byte | int | int32 | int64 | string](elements []T) string {
	var buffer bytes.Buffer
	buffer.WriteString("{")
	length := len(elements)
	if length > 0 {
		buffer.WriteString(fmt.Sprintf("%v", elements[0]))
		for i := 1; i < length; i++ {
			buffer.WriteString(fmt.Sprintf(", %v", elements[i]))
		}
	}
	buffer.WriteString("}")
	return buffer.String()
}
