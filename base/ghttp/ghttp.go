package ghttp

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

// // ====================================================================================================
// // Request & Response
// // ====================================================================================================
// type R2 struct {
// 	id   int32
// 	Next *R2
// 	//////////////////////////////////////////////////
// 	// 0: 讀取第一行, 1: 讀取 Header, 2: 讀取 Data, 3: 等待數據寫出(Response) 4. 完成數據複製到寫出緩存
// 	State int8
// 	*Request
// 	*Response
// 	Header

// 	// 讀取長度
// 	ReadLength int32

// 	// Body 數據
// 	Body       []byte
// 	BodyLength int32
// }

// func NewR2(id int32) *R2 {
// 	rr := &R2{
// 		id:         id,
// 		Next:       nil,
// 		State:      0,
// 		Header:     map[string][]string{},
// 		Body:       make([]byte, 64*1024),
// 		BodyLength: 0,
// 	}
// 	rr.Request = newRequest(rr)
// 	rr.Response = newResponse(rr)
// 	return rr
// }

// func (rr *R2) GetId() int32 {
// 	return rr.id
// }

// func (rr *R2) Add(r2 *R2) {
// 	r := rr
// 	for r.Next != nil {
// 		r = r.Next
// 	}
// 	r.Next = r2
// }

// func (rr *R2) HasLineData(buffer *[]byte, i int32, o int32, length int32) bool {
// 	// fmt.Printf("(rr *R2) HasLineData | i: %d, o: %d, length: %d\n", i, o, length)

// 	if length == 0 {
// 		return false
// 	}

// 	rr.ReadLength = 0
// 	value := -1

// 	if o < i {
// 		// fmt.Printf("(rr *R2) HasLineData | buffer0: %+v\n", (*buffer)[o:i])
// 		value = bytes.IndexByte((*buffer)[o:i], '\n')
// 		// fmt.Printf("(rr *R2) HasLineData | value(o < i): %d\n", value)

// 	} else {
// 		value = bytes.IndexByte((*buffer)[o:], '\n')
// 		// fmt.Printf("(rr *R2) HasLineData | buffer1: %+v\n", (*buffer)[o:])

// 		if value != -1 {
// 			rr.ReadLength = int32(value) + 1
// 			// fmt.Printf("(rr *R2) HasLineData | value([o:]): %d\n", value)
// 			return true
// 		}

// 		rr.ReadLength = int32(len((*buffer)[o:]))
// 		// fmt.Printf("(rr *R2) HasLineData | temp ReadLength: %d\n", rr.ReadLength)
// 		value = bytes.IndexByte((*buffer)[:i], '\n')
// 		// fmt.Printf("(rr *R2) HasLineData | buffer2: %+v\n", (*buffer)[:i])
// 		// fmt.Printf("(rr *R2) HasLineData | value([:i]): %d\n", value)
// 	}

// 	if value != -1 {
// 		rr.ReadLength += int32(value) + 1
// 		// fmt.Printf("(rr *R2) HasLineData | value: %d\n", value)
// 		return true
// 	}

// 	rr.ReadLength = 0
// 	return false
// }

// func (rr *R2) HasEnoughData(buffer *[]byte, i int32, o int32, length int32) bool {
// 	fmt.Printf("(rr *R2) HasEnoughData | length: %d, ReadLength: %d\n", length, rr.ReadLength)
// 	return length >= rr.ReadLength
// }

// func (rr *R2) SetHeader(key string, value string) {
// 	if _, ok := rr.Header[key]; !ok {
// 		rr.Header[key] = []string{value}
// 	} else {
// 		rr.Header[key] = append(rr.Header[key], value)
// 	}
// }

// // 生成 Response message
// func (rr *R2) FormResponse() []byte {
// 	var buffer bytes.Buffer
// 	// HTTP/1.1 200 OK\r\n
// 	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n", rr.Proto, rr.Code, rr.Message))

// 	// Header
// 	for k, v := range rr.Header {
// 		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
// 	}

// 	if _, ok := rr.Header["Content-Length"]; ok {
// 		buffer.WriteString("\r\n")
// 		buffer.Write(rr.Body)
// 	}

// 	return buffer.Bytes()
// }

// func (rr *R2) setBodyLength() {
// 	rr.Header["Content-Length"] = []string{strconv.Itoa(len(rr.Body))}
// }

// // ====================================================================================================
// // Request
// // ====================================================================================================
// type Request struct {
// 	*R2
// 	Method string
// 	Query  string
// 	//
// 	Proto  string
// 	Params map[string]string
// }

// func NewRequest(method string, uri string, params map[string]string) (*Request, error) {
// 	r2 := NewR2(-1)
// 	r2.Request.Method = method
// 	r2.Request.Proto = "HTTP/1.1"
// 	r2.Request.Params = params

// 	var host string
// 	var ok bool
// 	host, r2.Query, ok = strings.Cut(uri, "/")
// 	if ok {
// 		r2.Query = fmt.Sprintf("/%s", r2.Query)
// 		fmt.Printf("NewRequest | Query: %s\n", r2.Query)
// 	}
// 	r2.Header["Host"] = []string{host}
// 	return r2.Request, nil
// }

// func newRequest(r2 *R2) *Request {
// 	r := &Request{
// 		R2:     r2,
// 		Proto:  "HTTP/1.1",
// 		Params: map[string]string{},
// 	}
// 	return r
// }

// // 解析第一行數據
// // parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
// func (r *Request) ParseFirstLine(line string) bool {
// 	var ok bool
// 	r.Method, r.Query, ok = strings.Cut(line, " ")

// 	if !ok {
// 		return false
// 	}

// 	r.Query, r.Proto, ok = strings.Cut(r.Query, " ")

// 	if !ok {
// 		return false
// 	}

// 	r.Query = strings.TrimPrefix(r.Query, "?")

// 	fmt.Printf("(r *Request) ParseFirstLine | Method: %s, Query: %s, Proto: %s\n", r.Method, r.Query, r.Proto)
// 	return true
// }

// // 解析第一行數據中的請求路徑
// func (r *Request) ParseQuery() (bool, error) {
// 	var ok bool
// 	var params string
// 	r.Query, params, ok = strings.Cut(r.Query, "?")

// 	if !ok {
// 		return false, nil
// 	}

// 	fmt.Printf("(r *Request) ParseQuery | Query: %s, params: %s\n", r.Query, params)
// 	err := r.ParseParams(params)

// 	if err != nil {
// 		return true, errors.Wrapf(err, "Failed to parse params: %s", params)
// 	}

// 	fmt.Printf("(r *Request) ParseQuery | params: %+v\n", r.Params)
// 	return true, nil
// }

// // 解析第一行數據中的請求路徑中的 GET 參數
// func (r *Request) ParseParams(params string) error {
// 	var key, value string
// 	var ok bool
// 	var err error

// 	for params != "" {
// 		key, params, _ = strings.Cut(params, "&")

// 		if strings.Contains(key, ";") {
// 			fmt.Printf("(r *Request) ParseParams | invalid semicolon separator in query(%s)\n", key)
// 			continue
// 		}

// 		if key == "" {
// 			fmt.Printf("(r *Request) ParseParams | Empty query is found.\n")
// 			continue
// 		}

// 		key, value, ok = strings.Cut(key, "=")

// 		if !ok {
// 			continue
// 		}

// 		// 將 url 上的參數加入 params 管理
// 		r.Params[key] = value
// 	}

// 	return err
// }

// func (r Request) GetParam(key string) (bool, string) {
// 	if param, ok := r.Params[key]; ok {
// 		return true, param
// 	}
// 	return false, ""
// }

// func (r Request) FormRequest() []byte {
// 	// Accept: */*

// 	var buffer bytes.Buffer
// 	// GET /end HTTP/1.1
// 	buffer.WriteString(fmt.Sprintf("%s %s %s\r\n", r.Method, r.Query, r.Proto))

// 	// Header
// 	/*
// 				Content-Type: application/json
// 				User-Agent: PostmanRuntime/7.29.2
// 				Accept:
// 				Postman-Token: 6746eca0-5849-4c5f-a208-2d981c6100ff
// 				Host: 192.168.0.198:3333
// 				Accept-Encoding: gzip, deflate, br
// 				Connection: keep-alive
// 				Content-Length: 35

// 				{
// 					"id":0,
// 					"msg":"test"
// 				}

// 				key: User-Agent, value: Go-http-client/1.1
// 		(a *HttpAnser) Read | Header, key: Accept-Encoding, value: gzip
// 	*/

// 	// r.R2.Header["Content-Type"] = []string{"application/json"}
// 	r.R2.Header["User-Agent"] = []string{"Go-http-client/1.1"}
// 	// r.R2.Header["Accept"] = []string{"*/*"}
// 	// r.R2.Header["Postman-Token"] = []string{"6746eca0-5849-4c5f-a208-2d981c6100ff"}
// 	r.R2.Header["Accept-Encoding"] = []string{"gzip"}
// 	// r.R2.Header["Connection"] = []string{"keep-alive"}

// 	for k, v := range r.R2.Header {
// 		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
// 	}

// 	buffer.WriteString("\r\n")

// 	if _, ok := r.R2.Header["Content-Length"]; ok {
// 		buffer.WriteString("\r\n")
// 		buffer.Write(r.R2.Body)
// 	}
// 	result := buffer.Bytes()
// 	fmt.Printf("(r Request) FormRequest | result: %s\n", string(result))
// 	return result
// }

// func (r *Request) Json(obj any) {
// 	r.Header["Content-Type"] = jsonContentType
// 	r.Body, _ = json.Marshal(obj)
// 	r.setBodyLength()
// }

// // ====================================================================================================
// // Response
// // ====================================================================================================
// type Response struct {
// 	*R2
// 	Code    int32
// 	Message string
// }

// func newResponse(r2 *R2) *Response {
// 	r := &Response{
// 		R2: r2,
// 	}
// 	return r
// }

// // Status sets the HTTP response code.
// func (r *Response) Status(code int32) {
// 	r.Code = code
// 	r.Message = StatusText(code)
// }

// func (r *Response) Json(code int32, obj any) {
// 	r.Status(code)

// 	for k := range r.R2.Header {
// 		delete(r.R2.Header, k)
// 	}

// 	r.R2.Header["Content-Type"] = jsonContentType
// 	r.Body, _ = json.Marshal(obj)
// 	r.setBodyLength()
// }

// // 解析第一行數據
// // parseRequestLine parses "HTTP/1.1 200 OK" into its three parts.
// func (r *Response) ParseFirstLine(line string) bool {
// 	var ok bool
// 	r.Proto, r.Message, ok = strings.Cut(line, " ")

// 	if !ok {
// 		return false
// 	}

// 	var codeString string
// 	codeString, r.Message, ok = strings.Cut(r.Message, " ")

// 	if !ok {
// 		r.Message = codeString
// 		return false
// 	}

// 	code, err := strconv.Atoi(codeString)

// 	if err != nil {
// 		return false
// 	}

// 	r.Code = int32(code)
// 	fmt.Printf("(r *Response) ParseFirstLine | Proto: %s, Code: %d, Message: %s\n", r.Proto, r.Code, r.Message)
// 	return true
// }
