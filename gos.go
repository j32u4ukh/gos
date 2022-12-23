package gos

import (
	"fmt"
	"gos/ans"
	"gos/ask"
	"gos/define"
	"sync"

	"github.com/pkg/errors"
)

/* TODO:
1. 優化 Anser 與 Asker ◎
2. 優化 TransData，一開始初始化一大段記憶體空間，空間不足時，再以二次冪的規則動態增加 ◎
3. Asker 使用寫入緩存，當斷線重連的過程中需要寫出數據，會先寫到緩存，當連線物件建立後，再將數據傳出去 ◎
4. Anser 與 Asker 使用新版 TransData ◎
5. Anser 與 Asker 移到各自的套件當中 ◎
6. 新增 Socket type(目前是其中一種 TCP) ◎
7. 實作 HTTP Server
8. 將不同 Socket type 會有不同的部分抽象出來，允許根據需求抽換該部分
9. 實作 WebSocket Server
*/

var server *goserver
var once sync.Once

func init() {
	if server == nil {
		once.Do(func() {
			server = newGoserver()
		})
	}
}

func Listen(port int32, socketType define.SocketType) (*ans.Anser, error) {
	anser, err := server.listen(port, socketType)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to listen on port %d", port)
	}

	return anser, nil
}

// 開始所有已註冊的監聽
func StartListen() {
	var anser *ans.Anser
	for _, anser = range server.anserMap {
		go anser.Listen()
	}
}

func Bind(site int32, ip string, port int, socketType define.SocketType) (*ask.Asker, error) {
	asker, err := server.bind(site, ip, port, socketType)

	if err != nil {
		return nil, errors.Wrapf(err, "Failed to bind with %s:%d.", ip, port)
	}

	return asker, err
}

// 開始所有已註冊的監聽
func StartConnect() error {
	var asker *ask.Asker
	var err error

	for _, asker = range server.askerMap {
		err = asker.Connect()

		if err != nil {
			ip, port := asker.GetAddress()
			return errors.Wrapf(err, "Failed to connect to %s:%d.", ip, port)
		}
	}

	return nil
}

// 開始讀取數據與處理
func RunAns() {
	var anser *ans.Anser
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
	var asker *ask.Asker
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

		fmt.Printf("SendToServer | Send to site: %d, length: %d, data: %+v\n", site, length, (*data)[:length])

		return nil
	}
	return errors.New(fmt.Sprintf("Unknown site: %d", site))
}
