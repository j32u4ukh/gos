package ghttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/j32u4ukh/gos/utils"
	"github.com/pkg/errors"
)

// 工作完成時的 Callback 函式
type HandlerFunc func(*Context)
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
	// 完成數據複製到寫出緩存
	FINISH_RESPONSE
)

func (cs ContextState) String() string {
	switch cs {
	case READ_FIRST_LINE:
		return "READ_FIRST_LINE"
	case READ_HEADER:
		return "READ_HEADER"
	case READ_BODY:
		return "READ_BODY"
	case WRITE_RESPONSE:
		return "WRITE_RESPONSE"
	case FINISH_RESPONSE:
		return "FINISH_RESPONSE"
	default:
		return "Unknown ContextState"
	}
}

/*
Request 和 Response 的 Header 應該分開來，因為端點函式中既會讀取 Request 的 Header，也會寫出 Response 的 Header。
透過不同函式來讀寫即可，記得保留 Context 送出 Request 的能力。
*/
type Context struct {
	// Context 唯一碼
	id int32
	// 對應 連線結構 的 id
	Cid int32
	// 對應 工作結構 的 id
	Wid int32
	// 工作流程當前階段
	State ContextState
	*Request
	*Response
}

func NewContext(id int32) *Context {
	c := &Context{
		id:       id,
		Cid:      -1,
		Wid:      -1,
		State:    READ_FIRST_LINE,
		Request:  newRequest(),
		Response: newResponse(),
	}
	return c
}

func (c *Context) GetId() int32 {
	return c.id
}

func (c *Context) Json(code int32, obj any) {
	c.Response.Json(code, obj)
}

func (c Context) ReadJson(obj any) error {
	if c.Request.BodyLength > 0 {
		data := c.Request.Body[:c.Request.BodyLength]
		err := json.Unmarshal(data, obj)
		if err != nil {
			return errors.Wrap(err, "Failed to unmarshal body to json.")
		}
	}
	return nil
}

func (c Context) ReadBytes() []byte {
	if c.Request.BodyLength > 0 {
		result := make([]byte, c.Request.BodyLength)
		copy(result, c.Request.Body[:c.Request.BodyLength])
		return result
	}
	return nil
}

func (c *Context) Release() {
	c.Cid = -1
	c.Wid = -1
	c.State = READ_FIRST_LINE
	c.Request.Release()
	c.Response.Release()
}

// ====================================================================================================
// Request
// ====================================================================================================
type Request struct {
	// ex: GET
	Method string
	// ex: /user/get
	Query string
	// ex: HTTP/1.1
	Proto  string
	Params map[string]string
	Values map[string]any

	Header

	// 讀取長度
	ReadLength int32

	// Body 數據
	// NOTE: Body 和 BodyLength 之後將改為私有變數，要存取的話需透過函式來操作。
	Body       []byte
	BodyLength int32
}

func NewRequest(method string, uri string, params map[string]string) (*Request, error) {
	c := NewContext(-1)
	c.Request.FormRequest(method, uri, params)
	return c.Request, nil
}

func newRequest() *Request {
	r := &Request{
		Proto:      "HTTP/1.1",
		Params:     map[string]string{},
		Values:     make(map[string]any),
		Header:     make(Header),
		ReadLength: 0,
		Body:       make([]byte, 64*1024),
		BodyLength: 0,
	}
	return r
}

func (r *Request) FormRequest(method string, uri string, params map[string]string) {
	r.Method = method
	r.Proto = "HTTP/1.1"
	r.Params = params
	for key, value := range params {
		r.Values[key] = value
	}
	host, query, ok := strings.Cut(uri, "/")
	if ok {
		r.Query = fmt.Sprintf("/%s", query)
		utils.Debug("Query: %s", r.Query)
	}
	r.Header["Host"] = []string{host}
}

// 檢查是否有一行數據(以換行符 '\n' 來區分)
func (r *Request) HasLineData(buffer *[]byte, i int32, o int32, length int32) bool {
	// fmt.Printf("(c *Context) HasLineData | i: %d, o: %d, length: %d\n", i, o, length)

	if length == 0 {
		return false
	}

	r.ReadLength = 0
	value := -1

	if o < i {
		// fmt.Printf("(c *Context) HasLineData | buffer0: %+v\n", (*buffer)[o:i])
		value = bytes.IndexByte((*buffer)[o:i], '\n')
		// fmt.Printf("(c *Context) HasLineData | value(o < i): %d\n", value)

	} else {
		value = bytes.IndexByte((*buffer)[o:], '\n')
		// fmt.Printf("(c *Context) HasLineData | buffer1: %+v\n", (*buffer)[o:])

		if value != -1 {
			r.ReadLength = int32(value) + 1
			// fmt.Printf("(c *Context) HasLineData | value([o:]): %d\n", value)
			return true
		}

		r.ReadLength = int32(len((*buffer)[o:]))
		// fmt.Printf("(c *Context) HasLineData | temp ReadLength: %d\n", c.ReadLength)
		value = bytes.IndexByte((*buffer)[:i], '\n')
		// fmt.Printf("(c *Context) HasLineData | buffer2: %+v\n", (*buffer)[:i])
		// fmt.Printf("(c *Context) HasLineData | value([:i]): %d\n", value)
	}

	if value != -1 {
		r.ReadLength += int32(value) + 1
		// fmt.Printf("(c *Context) HasLineData | value: %d\n", value)
		return true
	}

	r.ReadLength = 0
	return false
}

func (r *Request) HasEnoughData(buffer *[]byte, i int32, o int32, length int32) bool {
	utils.Debug("length: %d, ReadLength: %d", length, r.ReadLength)
	return length >= r.ReadLength
}

// 解析第一行數據
// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func (r *Request) ParseFirstReqLine(line string) bool {
	var ok bool
	r.Method, r.Query, ok = strings.Cut(line, " ")
	if !ok {
		return false
	}
	r.Query, r.Proto, ok = strings.Cut(r.Query, " ")
	if !ok {
		return false
	}
	r.Query = strings.TrimPrefix(r.Query, "?")
	utils.Debug("Method: %s, Query: %s, Proto: %s", r.Method, r.Query, r.Proto)
	return true
}

// 解析第一行數據中的請求路徑
func (r *Request) ParseQuery() (bool, error) {
	var ok bool
	var params string
	r.Query, params, ok = strings.Cut(r.Query, "?")
	if !ok {
		return false, nil
	}
	utils.Debug("Query: %s, params: %s", r.Query, params)
	err := r.ParseParams(params)

	if err != nil {
		return true, errors.Wrapf(err, "Failed to parse params: %s", params)
	}

	utils.Debug("params: %+v", r.Params)
	utils.Debug("values: %+v", r.Values)
	return true, nil
}

// 解析第一行數據中的請求路徑中的 GET 參數
func (r *Request) ParseParams(params string) error {
	var key, value string
	var ok bool
	var err error

	for params != "" {
		key, params, _ = strings.Cut(params, "&")

		if strings.Contains(key, ";") {
			// fmt.Printf("(r *Request) ParseParams | invalid semicolon separator in query(%s)\n", key)
			utils.Warn("invalid semicolon separator in query(%s)", key)
			continue
		}

		if key == "" {
			// fmt.Printf("(r *Request) ParseParams | Empty query is found.\n")
			utils.Warn("Empty query is found.")
			continue
		}

		key, value, ok = strings.Cut(key, "=")

		if !ok {
			continue
		}

		// 將 url 上的參數加入 params 管理
		r.Params[key] = value
	}

	return err
}

func (r Request) GetParam(key string) (bool, string) {
	if param, ok := r.Params[key]; ok {
		return true, param
	}
	return false, ""
}

func (r Request) GetValue(key string) any {
	if value, ok := r.Values[key]; ok {
		return value
	}
	return nil
}

func (r *Request) Json(obj any) {
	r.Header["Content-Type"] = jsonContentType
	data, _ := json.Marshal(obj)
	// r.BodyLength = int32(len(data))
	// copy(r.Body[:r.BodyLength], data[:r.BodyLength])
	r.SetBody(data, int32(len(data)))
	r.SetContentLength()
}

// 供 Request 設置 Body 數據
func (r *Request) SetBody(data []byte, length int32) {
	r.BodyLength = length
	copy(r.Body[:length], data[:length])
}

// 協助設置標頭檔的 Content-Length
func (r *Request) SetContentLength() {
	r.Header["Content-Length"] = []string{strconv.Itoa(int(r.BodyLength))}
}

func (r Request) ToRequestData() []byte {
	// Accept: */*

	var buffer bytes.Buffer
	// GET /end HTTP/1.1
	buffer.WriteString(fmt.Sprintf("%s %s %s\r\n", r.Method, r.Query, r.Proto))

	/*
		Content-Type: application/json
		User-Agent: Go-http-client/1.1
		Host: 192.168.0.198:3333
		Accept-Encoding: gzip
		Connection: keep-alive
		Content-Length: 35

		{
			"id":0,
			"msg":"test"
		}
	*/

	r.Header["User-Agent"] = []string{"Go-http-client/1.1"}
	r.Header["Accept-Encoding"] = []string{"gzip"}

	for k, v := range r.Header {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}

	buffer.WriteString("\r\n")

	if _, ok := r.Header["Content-Length"]; ok {
		buffer.WriteString("\r\n")
		buffer.Write(r.Body[:r.BodyLength])
	}
	result := buffer.Bytes()
	utils.Debug("result: %s", string(result))
	return result
}

func (r *Request) Release() {
	r.Method = ""
	r.Query = ""
	r.Proto = ""
	r.BodyLength = 0
	r.ReadLength = 0
	var key string
	for key = range r.Params {
		delete(r.Params, key)
	}
	for key = range r.Values {
		delete(r.Values, key)
	}
	for key = range r.Header {
		delete(r.Header, key)
	}
}

// ====================================================================================================
// Response
// ====================================================================================================
type Response struct {
	Code    int32
	Message string
	Proto   string

	Header

	// 讀取長度
	ReadLength int32

	// Body 數據
	// NOTE: Body 和 BodyLength 之後將改為私有變數，要存取的話需透過函式來操作。
	Body       []byte
	BodyLength int32
}

func newResponse() *Response {
	r := &Response{
		Code:       -1,
		Message:    "",
		Proto:      "HTTP/1.1",
		Header:     make(Header),
		ReadLength: 0,
		Body:       make([]byte, 64*1024),
		BodyLength: 0,
	}
	return r
}

// 解析第一行數據
// parseRequestLine parses "HTTP/1.1 200 OK" into its three parts.
func (r *Response) ParseFirstResLine(line string) bool {
	var ok bool
	r.Proto, r.Message, ok = strings.Cut(line, " ")

	if !ok {
		return false
	}

	var codeString string
	codeString, r.Message, ok = strings.Cut(r.Message, " ")

	if !ok {
		r.Message = codeString
		return false
	}

	code, err := strconv.Atoi(codeString)

	if err != nil {
		return false
	}

	r.Code = int32(code)
	utils.Debug("Proto: %s, Code: %d, Message: %s", r.Proto, r.Code, r.Message)
	return true
}

func (r *Response) SetHeader(key string, value string) {
	if _, ok := r.Header[key]; !ok {
		r.Header[key] = []string{value}
	} else {
		r.Header[key] = append(r.Header[key], value)
	}
}

// Status sets the HTTP response code.
func (r *Response) Status(code int32) {
	r.Code = code
	r.Message = StatusText(code)
}

func (r *Response) Json(code int32, obj any) {
	r.Status(code)

	for k := range r.Header {
		delete(r.Header, k)
	}

	r.Header["Content-Type"] = jsonContentType
	data, _ := json.Marshal(obj)
	// r.BodyLength = int32(len(data))
	// copy(r.Body[:r.BodyLength], data[:r.BodyLength])
	r.SetBody(data, int32(len(data)))
	r.SetContentLength()
}

// 供 Request 設置 Body 數據
func (r *Response) SetBody(data []byte, length int32) {
	r.BodyLength = length
	copy(r.Body[:length], data[:length])
}

func (r *Response) SetContentLength() {
	r.Header["Content-Length"] = []string{strconv.Itoa(int(r.BodyLength))}
}

// 生成 Response message
func (r Response) ToResponseData() []byte {
	var buffer bytes.Buffer
	// HTTP/1.1 200 OK\r\n
	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n", r.Proto, r.Code, r.Message))

	// Header
	for k, v := range r.Header {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}

	if _, ok := r.Header["Content-Length"]; ok {
		buffer.WriteString("\r\n")
		buffer.Write(r.Body[:r.BodyLength])
	}

	return buffer.Bytes()
}

func (r *Response) Release() {
	r.Code = -1
	r.Message = ""
	r.ReadLength = 0
	r.BodyLength = 0

	for k := range r.Header {
		delete(r.Header, k)
	}
}
