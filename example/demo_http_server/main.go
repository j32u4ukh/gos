package main

import (
	"fmt"
	"gos"
	"gos/ans"
	"gos/ask"
	"gos/base/ghttp"
	"gos/define"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"time"
)

/*
網路多媒體資源需表示為典範形式(canonical form)，一般來說透過HTTP傳輸的Entity-Body在傳送階段之前必須被表示為合法的媒體典範形式，
像是在Content-Encoding編碼階段前Entity-Body必須符合典範形式；

像是"text"的多媒體形式利用CRLF當作文字的斷行，然而在HTTP傳送Entity Body時會單獨使用CR或LF代表斷行；
所以HTTP的應用必須識別CR/LF/CRLF為合法的斷行；
此外如果字符集不是用八進位的13/10代表CR/LF的話，HTTP允許使用個別字符集定義的CR/LF當作斷行字元，但這樣的彈性只限於Entity Body，
但斷行字元不能用來取代CRLF在HTTP的控制結構(如headers/multipart-boundaries)

如果"charset"沒有特別定義則與設為ISO-8859-1。

＊＊＊

在Request有Body但沒有Content-Length情況下，如果Server不能識別或計算Body的長度，則必須返回400(Bad Request)

＊＊＊

Keep-Alive ？？

＊＊＊

HTTP 1.1預設建立持久化連線，多個HTTP Request/Response可以復用同一個連線；
可以透過 close 選項表明此次Request/Response後就關閉連線。

// Determine whether to hang up after sending a request and body, or
// receiving a response and body
// 'header' is the request headers.
func shouldClose(major, minor int, header Header, removeCloseHeader bool) bool {
	if major < 1 {
		return true
	}

	conv := header["Connection"]
	hasClose := httpguts.HeaderValuesContainsToken(conv, "close")
	if major == 1 && minor == 0 {
		return hasClose || !httpguts.HeaderValuesContainsToken(conv, "keep-alive")
	}

	if hasClose && removeCloseHeader {
		header.Del("Connection")
	}

	return hasClose
}

＊＊＊

	10: '\n'
	13: '\r'

＊＊＊

*/

func main() {
	service_type := os.Args[1]
	var port int = 1023
	// if len(os.Args) >= 3 {
	// 	port, _ = strconv.Atoi(os.Args[2])
	// }

	if service_type == "ans" {
		RunAns(port)
	} else if service_type == "ask" {
		RunAsk("127.0.0.1", port)
	} else if service_type == "nr" {
		DemoNativeHttpRequest(port)
	} else if service_type == "ns" {
		DemoNativeHttpServer("192.168.0.198", port)
	}

	fmt.Println("[Example] Run | End of gos example.")
}

func RunAns(port int) {
	anser, err := gos.Listen(define.Http, int32(port))
	fmt.Printf("RunAns | Listen to port %d\n", port)

	if err != nil {
		fmt.Printf("Error: %+v\n", err)
		return
	}

	httpAnswer := anser.(*ans.HttpAnser)
	mrg := &Mgr{}
	mrg.Handler(httpAnswer.Router)

	fmt.Printf("(s *Service) RunAns | 伺服器初始化完成\n")
	gos.StartListen()
	fmt.Printf("(s *Service) RunAns | 開始監聽\n")
	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	for {
		start = time.Now()

		gos.RunAns()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}

func RunAsk(ip string, port int) {
	// demoNativeHttpRequest(ip, port)
	asker, err := gos.Bind(0, ip, port, define.Http)

	if err != nil {
		fmt.Printf("BindError: %+v\n", err)
		return
	}

	http := asker.(*ask.HttpAsker)
	fmt.Printf("http: %+v\n", http)

	req, err := ghttp.NewRequest(ghttp.MethodGet, "127.0.0.1:1023/abc/get", nil)

	if err != nil {
		fmt.Printf("NewRequestError: %+v\n", err)
		return
	}

	fmt.Printf("req: %+v\n", req)
	var site int32
	site, err = gos.SendRequest2(req, func(c *ghttp.Context) {
		fmt.Printf("I'm Context, Query: %s\n", c.Query)
	})

	fmt.Printf("site: %d\n", site)

	if err != nil {
		fmt.Printf("SendRequestError: %+v\n", err)
		return
	}

	var start time.Time
	var during, frameTime time.Duration = 0, 200 * time.Millisecond

	for {
		start = time.Now()

		gos.RunAsk()

		during = time.Since(start)
		if during < frameTime {
			time.Sleep(frameTime - during)
		}
	}
}

func DemoNativeHttpRequest(port int) {
	// client := &http.Client{}

	// //這邊可以任意變換 http method  GET、POST、PUT、DELETE
	// req, err := http.NewRequest("GET", fmt.Sprintf("%s:%d/abc/get", ip, port), nil)
	// if err != nil {
	// 	fmt.Println(err)
	// 	return
	// }
	// // req.Header.Add("If-None-Match", `W/"wyzzy"`)
	// res, err := client.Do(req)
	// requestURL := fmt.Sprintf("http://localhost:%d/abc/get", port)

	requestURL := fmt.Sprintf("http://127.0.0.1:%d/abc/get", port)

	res, err := http.Get(requestURL)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer res.Body.Close()
	sitemap, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Printf("err: %+v\n", err)
		return
	}

	fmt.Printf("%s\n", sitemap)
}

func DemoNativeHttpServer(ip string, port int) {
	// Listen for incoming connections
	l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		fmt.Println("Error listening:", err.Error())
		os.Exit(1)
	}

	// Close the listener when the application closes.
	defer l.Close()

	fmt.Println("Start listening...")
	for {
		// Listen for an incoming connection.
		conn, err := l.Accept()
		if err != nil {
			fmt.Println("Error accepting: ", err.Error())
			os.Exit(1)
		}
		// Handle connections
		go handleRequest(conn)
	}
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
	fmt.Println(buf)
	fmt.Println(request)

	fmt.Println()
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
	r := []byte("HTTP/1.1 200 OK\r\nConnection: close\r\nContent-Type: text/html\r\nContent-Length: 19\r\n\r\n<h1>Hola Mundo</h1>")
	conn.Write(r)
	// Close the connection when you're done with it.
	conn.Close()
}
