package gos

import (
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/j32u4ukh/glog/v2"
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

// 指定要監聽的 port，並生成 Anser 物件
func Listen(socketType define.SocketType, port int32) (ans.IAnswer, error) {
	if _, ok := server.anserMap[port]; !ok {
		laddr, _ := net.ResolveTCPAddr("tcp", fmt.Sprintf(":%d", port))
		anser, err := ans.NewAnser(socketType, laddr, 10, 10)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to listen on port %d.", port)
		}
		server.anserMap[port] = anser
	}
	return server.anserMap[port], nil
}

// 開始所有已註冊的監聽
func StartListen() {
	var anser ans.IAnswer
	for _, anser = range server.anserMap {
		go anser.Listen()
	}
}

// 向位置 ip:port 送出連線請求，利用 serverId 來識別多個連線
// serverId: server id
// ip: server ip
// port: server port
// socketType: 協定類型
func Bind(serverId int32, ip string, port int, socketType define.SocketType, onEvents base.OnEventsFunc, introduction *[]byte, heartbeat *[]byte) (ask.IAsker, error) {
	if _, ok := server.askerMap[serverId]; !ok {
		laddr := &net.TCPAddr{IP: net.ParseIP(ip), Port: port, Zone: ""}
		asker, err := ask.NewAsker(socketType, serverId, laddr, 10, onEvents, introduction, heartbeat)
		if err != nil {
			return nil, errors.Wrapf(err, "Failed to create an Asker for %s:%d.", ip, port)
		}
		server.askerMap[serverId] = asker
	}
	return server.askerMap[serverId], nil
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

func Run(run func()) {
	var anser ans.IAnswer
	var asker ask.IAsker
	var start time.Time
	var during time.Duration

	for {
		start = time.Now()

		// 處理各個 anser 讀取到的數據
		for _, anser = range server.anserMap {
			anser.Handler()
		}

		// 處理各個 asker 讀取到的數據
		for _, asker = range server.askerMap {
			asker.Handler()
		}

		// 外部定義的處理函式
		if run != nil {
			run()
		}

		during = time.Since(start)
		if during < server.frameTime {
			time.Sleep(server.frameTime - during)
		}
	}
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
	data := td.FormData()
	err := SendToServer(serverId, &data, int32(len(data)))
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
		// utils.Info("Send to site: %d, length: %d, data: %+v", serverId, length, (*data)[:length])
		return nil
	}
	return errors.New(fmt.Sprintf("Unknown site: %d", serverId))
}

// 傳送 http 訊息
func SendRequest(req *ghttp.Request, callback func(*ghttp.Context)) (int32, error) {
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
		ip, p, _ := strings.Cut(host[0], ":")
		var asker ask.IAsker
		var err error

		port, _ := strconv.Atoi(p)
		asker, err = Bind(server.nextServerId, ip, port, define.Http, nil, nil, nil)
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

func Disconnect(port int32, cid int32) error {
	var err error = nil
	if anser, ok := server.anserMap[port]; ok {
		err = anser.Disconnect(cid)
		if err != nil {
			return errors.Wrapf(err, "Failed to disconnect connection: %d-%d", port, cid)
		}
	} else {
		err = errors.Errorf("Not found anser for %d", port)
	}
	return err
}

func SetFrameTime(frameTime time.Duration) {
	server.frameTime = frameTime
}

func GetFrameTime() time.Duration {
	return server.frameTime
}

func SetLogger(lg *glog.Logger) {
	utils.SetLogger(lg)
}
