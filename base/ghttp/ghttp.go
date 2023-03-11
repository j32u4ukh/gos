package ghttp

import "github.com/j32u4ukh/glog"

// A Header represents the key-value pairs in an HTTP header.
//
// The keys should be in canonical form, as returned by
// CanonicalHeaderKey.
//
// Connection: 當 client 和 server 通信時對於長鏈接如何進行處理
// [Request]
// - close（告訴WEB服務器或者代理服務器，在完成本次請求的響應後，斷開連接，不要等待本次連接的後續請求了）。
// - keepalive（告訴WEB服務器或者代理服務器，在完成本次請求的響應後，保持連接，等待本次連接的後續請求）。
// [Request]
// - close（連接已經關閉）。
// - keepalive（連接保持着，在等待本次連接的後續請求）。 Keep-Alive：如果瀏覽器請求保持連接，則該頭部表明希望 WEB 服務器保持連接多長時間（秒）。例如：Keep-Alive：300
// Content-Type: WEB 服務器告訴瀏覽器自己響應的對象的類型。
// - text/html
// - text/html; charset=utf-8
// - application/json
// Content-Length: WEB 服務器告訴瀏覽器自己響應的對象的長度。若有 Data 數據，需描述數據長度。
//
// User-Agent: 告訴網站它是透過什麼工具（通過UA分析出瀏覽器名稱、瀏覽器版本號、渲染引擎、操作系統）發送請求的
// Mozilla/[version] ([system and browser information]) [platform] ([platform details]) [extensions]
// Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_3) AppleWebKit/604.5.6 (KHTML, like Gecko) Version/11.0.3 Safari/604.5.6
// 表示使用 Safari 瀏覽器，瀏覽器版本 11.0.3，網頁渲染引擎 WebKit 604.5.6，電腦操作系統 Mac OS。
// Mozilla/5.0 是一個通用標記符號，用來表示與 Mozilla 相容，這幾乎是現代瀏覽器的標配。Gecko 排版引擎（頁面渲染引擎）

type Header map[string][]string

// A MIMEHeader represents a MIME-style header mapping keys to sets of values.
type MIMEHeader map[string][]string

type H map[string]any

const (
	MethodGet  = "GET"
	MethodPost = "POST"
	COLON      = ":"
)

// const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"

var (
	jsonContentType = []string{"application/json"}
)

var logger *glog.Logger

func init() {
	logger = glog.GetLogger("log", "gos", glog.DebugLevel, false)
	logger.SetOptions(glog.DefaultOption(true, true), glog.UtcOption(8))
}
