package gos

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/base/ghttp"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"
	"github.com/pkg/errors"
)

var server *goserver
var once sync.Once
var LOGGER byte

func Init() {
	if server == nil {
		once.Do(func() {
			server = newGoserver()
		})
	}
}

func Listen(socketType define.SocketType, port int32) (ans.IAnswer, error) {
	anser, err := server.listen(socketType, port)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to listen on port %d", port)
	}

	return anser, nil
}

// 開始所有已註冊的監聽
func StartListen() {
	var anser ans.IAnswer
	for _, anser = range server.anserMap {
		go anser.Listen()
	}
}

func Bind(site int32, ip string, port int, socketType define.SocketType) (ask.IAsker, error) {
	asker, err := server.bind(site, ip, port, socketType)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to bind with %s:%d.", ip, port)
	}

	return asker, err
}

// 開始所有已註冊的監聽
func StartConnect() error {
	var asker ask.IAsker
	var site int32
	var err error

	for site, asker = range server.askerMap {
		err = asker.Connect()

		if err != nil {
			ip, port := asker.GetAddress()
			return errors.Wrapf(err, "Failed to connect to %s:%d.", ip, port)
		}

		if server.nextSite < site {
			server.nextSite = site
		}
	}

	// 啟動後，最大的 site 值 + 1，作為動態建立 Asker 時的 site 值
	server.nextSite++
	return nil
}

// 開始讀取數據與處理
func RunAns() {
	var anser ans.IAnswer
	// 處理各個 anser 讀取到的數據
	for _, anser = range server.anserMap {
		anser.Handler()
	}
}

func SendToClient(port int32, cid int32, data *[]byte, length int32) error {
	if anser, ok := server.anserMap[port]; ok {
		err := anser.Write(cid, data, length)

		if err != nil {
			return errors.Wrap(err, "Failed to send to client.")
		}

		return nil
	}
	return errors.New(fmt.Sprintf("Hasn't listen to port %d", port))
}

func RunAsk() {
	var asker ask.IAsker
	// 處理各個 asker 讀取到的數據
	for _, asker = range server.askerMap {
		asker.Handler()
	}
}

func SendToServer(site int32, data *[]byte, length int32) error {
	if asker, ok := server.askerMap[site]; ok {
		err := asker.Write(data, length)

		if err != nil {
			return errors.Wrap(err, "Failed to send to client.")
		}

		// fmt.Printf("SendToServer | Send to site: %d, length: %d, data: %+v\n", site, length, (*data)[:length])
		utils.Info("Send to site: %d, length: %d, data: %+v", site, length, (*data)[:length])

		return nil
	}
	return errors.New(fmt.Sprintf("Unknown site: %d", site))
}

// 傳送 http 訊息
func SendRequest(req *ghttp.Request, callback func(*ghttp.Context)) (int32, error) {
	// fmt.Printf("SendRequest | Request: %+v\n", req)
	utils.Info("Request: %+v", req)
	var asker ask.IAsker
	var site int32

	// 檢查是否有相同 Address、已建立的 Asker
	for site, asker = range server.askerMap {
		ip, port := asker.GetAddress()
		host := fmt.Sprintf("%s/%d", ip, port)

		if host == req.Header["Host"][0] {
			httpAsker := asker.(*ask.HttpAsker)
			httpAsker.Send(req, callback)
			return site, nil
		}
	}

	if host, ok := req.Header["Host"]; ok {
		// fmt.Printf("SendRequest | host: %s\n", host[0])

		ip, p, _ := strings.Cut(host[0], ":")
		// fmt.Printf("SendRequest | ip: %s, port: %s\n", ip, p)
		// fmt.Printf("SendRequest | query: %s\n", req.Query)
		var asker ask.IAsker
		var err error

		port, _ := strconv.Atoi(p)
		asker, err = Bind(server.nextSite, ip, port, define.Http)
		defer func() { server.nextSite++ }()

		if err != nil {
			return -1, errors.Wrapf(err, "Failed to bind to host: %s", host[0])
		}

		httpAsker := asker.(*ask.HttpAsker)
		httpAsker.Send(req, callback)
		return server.nextSite, nil
	}

	return -1, errors.New("Request 中未定義 uri")
}
