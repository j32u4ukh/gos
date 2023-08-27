package ghttp

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

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
	// 和 GET 一樣，只是 HEAD 只會讀取 HTTP Header 的資料。
	MethodHead = "HEAD"
	// 讀取資料
	MethodGet = "GET"
	// 新增一項資料。（如果存在會新增一個新的）
	MethodPost = "POST"
	// 新增一項資料，如果存在就覆蓋過去（還是只有一筆資料）。因為是直接覆蓋，因此資料必須是完整的。
	// 如果沒有傳，則會被更新為空值。
	MethodPut = "PUT"
	// 附加新的資料在已經存在的資料後面（資料必須已經存在，PATCH 會擴充/更新這項資料）。
	// 類似 PUT 方法，但沒有資料的欄位則不會更新，可以更新結構中的一部份。
	MethodPatch = "PATCH"
	// 刪除資料。
	MethodDelete = "DELETE"
	// 返回伺服器支援的方法。
	MethodOptions = "OPTIONS"
	COLON         = ":"
	// ==================================================
	// CORS
	// ==================================================
	HeaderOrigin            = "Origin"
	HeaderCorsOrigin        = "Access-Control-Allow-Origin"
	HeaderCorsMaxAge        = "Access-Control-Max-Age"
	HeaderCorsMethods       = "Access-Control-Allow-Methods"
	HeaderCorsCredentials   = "Access-Control-Allow-Credentials"
	HeaderCorsAllowHeaders  = "Access-Control-Allow-Headers"
	HeaderCorsExposeHeaders = "Access-Control-Expose-Headers"
	// Header Name
	HeaderAccept          = "Accept"
	HeaderAcceptLanguage  = "Accept-Language"
	HeaderAcceptEncoding  = "Accept-Encoding"
	HeaderContentLanguage = "Content-Language"
	HeaderContentType     = "Content-Type"
	HeaderCacheControl    = "Cache-Control"
	HeaderDPR             = "DPR"
	HeaderDownlink        = "Downlink"
	HeaderExpires         = "Expires"
	HeaderLastModified    = "Last-Modified"
	HeaderPragma          = "Pragma"
	HeaderSaveData        = "Save-Data"
	HeaderViewportWidth   = "Viewport-Width"
	HeaderWidth           = "Width"
	HeaderHost            = "Host"
	HeaderContentLength   = "Content-Length"
	HeaderUserAgent       = "User-Agent"
)

var (
	jsonContentType = []string{"application/json"}
	titleCase       cases.Caser
)

func init() {
	titleCase = cases.Title(language.English)
}

func CapitalString(input string) string {
	return titleCase.String(input)
}
