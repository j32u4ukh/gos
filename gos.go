package gos

import (
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/j32u4ukh/gos/ans"
	"github.com/j32u4ukh/gos/ask"
	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/base/ghttp"
	"github.com/j32u4ukh/gos/define"
	"github.com/j32u4ukh/gos/utils"
	"github.com/pkg/errors"
)

var server *goserver
var once sync.Once

func init() {
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

func Bind(site int32, ip string, port int, socketType define.SocketType, onEvents base.OnEventsFunc) (ask.IAsker, error) {
	asker, err := server.bind(site, ip, port, socketType, onEvents)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to bind with %s:%d.", ip, port)
	}

	return asker, err
}

// 開始所有已註冊的監聽
func StartConnect() error {
	var asker ask.IAsker
	var serverId int32
	var err error

	for serverId, asker = range server.askerMap {
		err = asker.Connect()

		if err != nil {
			ip, port := asker.GetAddress()
			return errors.Wrapf(err, "Failed to connect to %s:%d.", ip, port)
		}

		if server.nextServerId < serverId {
			server.nextServerId = serverId
		}
	}

	// 啟動後，最大的 site 值 + 1，作為動態建立 Asker 時的 site 值
	server.nextServerId++
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

func SendTransDataToServer(serverId int32, td *base.TransData) error {
	data := td.GetData()
	length := td.GetLength()
	err := SendToServer(serverId, &data, length)
	if err != nil {
		return errors.Wrap(err, "Failed to send transdata to server.")
	}
	return nil
}

func SendToServer(serverId int32, data *[]byte, length int32) error {
	if asker, ok := server.askerMap[serverId]; ok {
		err := asker.Write(data, length)

		if err != nil {
			return errors.Wrap(err, "Failed to send to server.")
		}

		// fmt.Printf("SendToServer | Send to site: %d, length: %d, data: %+v\n", site, length, (*data)[:length])
		utils.Info("Send to site: %d, length: %d, data: %+v", serverId, length, (*data)[:length])

		return nil
	}
	return errors.New(fmt.Sprintf("Unknown site: %d", serverId))
}

// 傳送 http 訊息
func SendRequest(req *ghttp.Request, callback func(*ghttp.Context)) (int32, error) {
	// fmt.Printf("SendRequest | Request: %+v\n", req)
	utils.Info("Request: %+v", req)
	var asker ask.IAsker
	var serverId int32

	// 檢查是否有相同 Address、已建立的 Asker
	for serverId, asker = range server.askerMap {
		ip, port := asker.GetAddress()
		host := fmt.Sprintf("%s/%d", ip, port)

		if host == req.Header["Host"][0] {
			httpAsker := asker.(*ask.HttpAsker)
			httpAsker.Send(req, callback)
			return serverId, nil
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
		asker, err = Bind(server.nextServerId, ip, port, define.Http, nil)
		defer func() { server.nextServerId++ }()

		if err != nil {
			return -1, errors.Wrapf(err, "Failed to bind to host: %s", host[0])
		}

		httpAsker := asker.(*ask.HttpAsker)
		httpAsker.Send(req, callback)
		return server.nextServerId, nil
	}

	return -1, errors.New("Request 中未定義 uri")
}
