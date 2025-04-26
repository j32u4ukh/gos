package ghttp

type HttpMode byte

const (
	HTTPMODE_REQUEST HttpMode = iota
	HTTPMODE_RESPONSE
)

// Header 表示 HTTP 標頭的鍵值對，鍵需為規範化格式（由 CanonicalHeaderKey 返回）。
//
// 常見 HTTP 標頭說明：
// - Connection：定義客戶端與服務器間長連接的處理方式。
//   [Request 請求]
//   - close：完成本次請求後斷開連接，不等待後續請求。
//   - keepalive：完成本次請求後保持連接，等待後續請求。
//   [Response 響應]
//   - close：表示連接已關閉。
//   - keepalive：表示連接保持，等待後續請求。Keep-Alive 標頭可指定保持連接的時間（秒），如 Keep-Alive: 300。
// - Content-Type：服務器告知瀏覽器響應內容的類型。
//   常見值：
//   - text/html
//   - text/html; charset=utf-8
//   - application/json
// - Content-Length：服務器告知瀏覽器響應內容的長度（字節）。若包含數據，需指定數據長度。
// - User-Agent：標識發送請求的工具，包含瀏覽器名稱、版本、渲染引擎及操作系統等資訊。
//   示例：Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_3) AppleWebKit/604.5.6 (KHTML, like Gecko) Version/11.0.3 Safari/604.5.6
//   說明：使用 Safari 瀏覽器（版本 11.0.3），渲染引擎為 WebKit 604.5.6，運行於 Mac OS。
//   註：Mozilla/5.0 是現代瀏覽器的通用標記，表示與 Mozilla 相容；Gecko 為常見的頁面渲染引擎。

// Header 表示 HTTP 協議的標頭鍵值對，鍵經過規範化處理，專為 HTTP 協議設計。
// MIMEHeader 表示 MIME 風格的標頭鍵值對，適用於更通用的場景，如電子郵件或 multipart 數據。
// 兩者區分是為了確保類型安全、語義清晰及支援特定協議的處理邏輯。
type Header map[string][]string
type MIMEHeader map[string][]string

type H map[string]any

const DEFAULT_HTTP_PROTO = "HTTP/1.1"

const (
	// 和 GET 一樣，只是 HEAD 只會讀取 HTTP Header 的資料。
	METHOD_HEAD = "HEAD"
	// 讀取資料
	METHOD_GET = "GET"
	// 新增一項資料。（如果存在會新增一個新的）
	METHOD_POST = "POST"
	// 新增一項資料，如果存在就覆蓋過去（還是只有一筆資料）。因為是直接覆蓋，因此資料必須是完整的。
	// 如果沒有傳，則會被更新為空值。
	METHOD_PUT = "PUT"
	// 附加新的資料在已經存在的資料後面（資料必須已經存在，PATCH 會擴充/更新這項資料）。
	// 類似 PUT 方法，但沒有資料的欄位則不會更新，可以更新結構中的一部份。
	METHOD_PATCH = "PATCH"
	// 刪除資料。
	METHOD_DELETE = "DELETE"
	// 返回伺服器支援的方法。
	METHOD_OPTIONS = "OPTIONS"
	COLON          = ":"
)

// Header 鍵值
const (
	HEADER_CONNECTION      = "Connection"
	HEADER_CONTENT_TYPE    = "Content-Type"
	HEADER_CONTENT_LENGTH  = "Content-Length"
	HEADER_USER_AGENT      = "User-Agent"
	HEADER_ACCEPT_ENCODING = "Accept-Encoding"
)

// Content 類型
const (
	CONTENT_TYPE_JSON = "application/json"
)

// Header of MethodOptions' response
// HTTP/1.1 200 OK
// Allow: GET, POST, HEAD, OPTIONS

// const errorHeaders = "\r\nContent-Type: text/plain; charset=utf-8\r\nConnection: close\r\n\r\n"

type HandlerFunc func(c *HttpContext)
type HandlerChain []HandlerFunc
type EndPointHandlers []*EndPoint

// 工作完成時的 Callback 函式
type ContextState int8

// HTTP 工作流程按照下方順序執行
const (
	// 讀取第一行
	READ_FIRST_LINE ContextState = iota
	// 讀取 Header
	READ_HEADER
	// 讀取 Data
	READ_BODY
	// 等待數據寫出(Response)
	WRITE_RESPONSE
	// 等待數據讀取(Response)
	READ_RESPONSE_FINISH
	// 完成數據複製到寫出緩存
	WRITE_FINISH_RESPONSE
)

func (cs ContextState) String() string {
	switch cs {
	case READ_FIRST_LINE:
		return "READ_FIRST_LINE"
	case READ_HEADER:
		return "READ_HEADER"
	case READ_BODY:
		return "READ_BODY"
	case READ_RESPONSE_FINISH:
		return "READ_RESPONSE"
	case WRITE_RESPONSE:
		return "WRITE_RESPONSE"
	case WRITE_FINISH_RESPONSE:
		return "FINISH_RESPONSE"
	default:
		return "Unknown ContextState"
	}
}
