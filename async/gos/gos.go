package gos

import (
	"sync"
	"time"

	"github.com/j32u4ukh/gos/async/gos/ans"
	"github.com/j32u4ukh/gos/async/gos/ask"
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

func RegisterAnser(port int32, anser ans.IAnser) {
	server.anserMap[port] = anser
}

func RegisterAsker(serverId int32, asker ask.IAsker) {
	server.askerMap[serverId] = asker
}

func Run(run func()) {
	var start time.Time
	var during time.Duration
	var anser ans.IAnser
	var asker ask.IAsker
	// 開始所有已註冊的監聽
	for _, anser = range server.anserMap {
		go anser.Listen()
	}
	for _, asker = range server.askerMap {
		go asker.Bind()
	}
	for {
		start = time.Now()
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

func Disconnect(port int32, cid int32) error {
	var err error = nil
	if anser, ok := server.anserMap[port]; ok {
		err = anser.Disconnect(cid, 3*time.Second)
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

// 向位置 ip:port 送出連線請求，利用 serverId 來識別多個連線
// serverId: server id
// ip: server ip
// port: server port
// socketType: 協定類型
// func Bind(serverId int32, ip string, port int, socketType define.SocketType, onEvents base.OnEventsFunc, introduction *[]byte, heartbeat *[]byte) (ask.IAsker, error) {
// 	if _, ok := server.askerMap[serverId]; !ok {
// 		laddr := &net.TCPAddr{IP: net.ParseIP(ip), Port: port, Zone: ""}
// 		asker, err := ask.NewAsker(
// 			socketType,
// 			serverId,
// 			laddr,
// 			utils.GosConfig.AskerWorkNumbers[socketType],
// 			onEvents,
// 			introduction,
// 			heartbeat,
// 		)
// 		if err != nil {
// 			return nil, errors.Wrapf(err, "Failed to create an Asker for %s:%d.", ip, port)
// 		}
// 		server.askerMap[serverId] = asker
// 	}
// 	return server.askerMap[serverId], nil
// }

// // 開始所有已註冊的監聽
// func StartConnect() error {
// 	var asker ask.IAsker
// 	var serverId int32
// 	var err error
// 	for serverId, asker = range server.askerMap {
// 		err = asker.Connect()
// 		if err != nil {
// 			ip, port := asker.GetAddress()
// 			return errors.Wrapf(err, "Failed to connect to %s:%d.", ip, port)
// 		}
// 		if server.nextServerId < serverId {
// 			server.nextServerId = serverId
// 		}
// 	}
// 	// 啟動後，最大的 site 值 + 1，作為動態建立 Asker 時的 site 值
// 	server.nextServerId++
// 	return nil
// }

// func SendTransDataToServer(serverId int32, td *base.TransData) error {
// 	data := td.FormData()
// 	err := SendToServer(serverId, &data, int32(len(data)))
// 	if err != nil {
// 		return errors.Wrap(err, "Failed to send transdata to server.")
// 	}
// 	return nil
// }

// func SendToServer(serverId int32, data *[]byte, length int32) error {
// 	if asker, ok := server.askerMap[serverId]; ok {
// 		err := asker.Write(data, length)
// 		if err != nil {
// 			return errors.Wrap(err, "Failed to send to server.")
// 		}
// 		// utils.Info("Send to site: %d, length: %d, data: %+v", serverId, length, (*data)[:length])
// 		return nil
// 	}
// 	return errors.New(fmt.Sprintf("Unknown site: %d", serverId))
// }

// // 傳送 http 訊息
// func SendRequest(req *ghttp.Request, callback func(*ghttp.Context)) (int32, error) {
// 	utils.Info("Request: %+v", req)
// 	var asker ask.IAsker
// 	var serverId int32
// 	// 檢查是否有相同 Address、已建立的 Asker
// 	for serverId, asker = range server.askerMap {
// 		ip, port := asker.GetAddress()
// 		host := fmt.Sprintf("%s/%d", ip, port)
// 		if host == req.Header[ghttp.HeaderHost][0] {
// 			httpAsker := asker.(*ask.HttpAsker)
// 			httpAsker.Send(req, callback)
// 			return serverId, nil
// 		}
// 	}
// 	if host, ok := req.Header[ghttp.HeaderHost]; ok {
// 		ip, p, _ := strings.Cut(host[0], ":")
// 		var asker ask.IAsker
// 		var err error
// 		port, _ := strconv.Atoi(p)
// 		asker, err = Bind(server.nextServerId, ip, port, define.Http, nil, nil, nil)
// 		defer func() { server.nextServerId++ }()
// 		if err != nil {
// 			return -1, errors.Wrapf(err, "Failed to bind to host: %s", host[0])
// 		}
// 		httpAsker := asker.(*ask.HttpAsker)
// 		httpAsker.Send(req, callback)
// 		return server.nextServerId, nil
// 	}
// 	return -1, errors.New("Request 中未定義 uri")
// }
