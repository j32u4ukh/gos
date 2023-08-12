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

type Context struct {
	// Context 唯一碼
	id int32
	// 對應 連線結構 的 id
	Cid int32
	// 對應 工作結構 的 id
	Wid int32
	//////////////////////////////////////////////////
	// 0: 讀取第一行, 1: 讀取 Header, 2: 讀取 Data, 3: 等待數據寫出(Response) 4. 完成數據複製到寫出緩存
	State int8
	Header
	*Request
	*Response

	// 讀取長度
	ReadLength int32

	// Body 數據
	// NOTE: Body 和 BodyLength 之後將改為私有變數，要存取的話需透過函式來操作。
	Body       []byte
	BodyLength int32
}

func NewContext(id int32) *Context {
	c := &Context{
		id:         id,
		Cid:        -1,
		Wid:        -1,
		State:      0,
		Header:     map[string][]string{},
		Body:       make([]byte, 64*1024),
		BodyLength: 0,
	}
	c.Request = newRequest(c)
	c.Response = newResponse(c)
	return c
}

func (c *Context) GetId() int32 {
	return c.id
}

func (c *Context) SetHeader(key string, value string) {
	if _, ok := c.Header[key]; !ok {
		c.Header[key] = []string{value}
	} else {
		c.Header[key] = append(c.Header[key], value)
	}
}

// 供 HTTP 套件設置 Body 數據
func (c *Context) SetBody(data []byte, length int32) {
	c.BodyLength = length
	copy(c.Body[:length], data[:length])
}

// 協助設置標頭檔的 Content-Length
func (c *Context) SetContentLength() {
	c.Header["Content-Length"] = []string{strconv.Itoa(int(c.BodyLength))}
}

func (c *Context) Json(code int32, obj any) {
	c.Response.Json(code, obj)
}

func (c *Context) ReadJson(obj any) error {
	if c.BodyLength > 0 {
		data := c.Body[:c.BodyLength]
		err := json.Unmarshal(data, obj)
		if err != nil {
			return errors.Wrap(err, "Failed to unmarshal body to json.")
		}
	}
	return nil
}

func (c *Context) ReadBytes() []byte {
	if c.BodyLength > 0 {
		result := make([]byte, c.BodyLength)
		copy(result, c.Body[:c.BodyLength])
		return result
	}
	return nil
}

func (c *Context) Release() {
	c.Cid = -1
	c.Wid = -1
	c.State = 0
	for key := range c.Header {
		delete(c.Header, key)
	}
	c.Request.Release()
	c.Response.Release()

	// 讀取長度
	c.ReadLength = 0
	c.BodyLength = 0
}

// ====================================================================================================
// Request
// ====================================================================================================
type Request struct {
	*Context
	// ex: GET
	Method string
	// ex: /user/get
	Query string
	// ex: HTTP/1.1
	Proto  string
	Params map[string]string
	Values map[string]any
}

func NewRequest(method string, uri string, params map[string]string) (*Request, error) {
	c := NewContext(-1)
	c.Method = method
	c.Proto = "HTTP/1.1"
	c.Params = params
	if c.Values == nil {
		c.Values = make(map[string]any)
	}
	for key, value := range params {
		c.Values[key] = value
	}

	var host string
	var ok bool
	host, c.Query, ok = strings.Cut(uri, "/")
	if ok {
		c.Query = fmt.Sprintf("/%s", c.Query)
		// fmt.Printf("NewRequest | Query: %s\n", c.Query)
		utils.Debug("Query: %s", c.Query)
	}
	c.Header["Host"] = []string{host}
	return c.Request, nil
}

func newRequest(c *Context) *Request {
	r := &Request{
		Context: c,
		Proto:   "HTTP/1.1",
		Params:  map[string]string{},
		Values:  make(map[string]any),
	}
	return r
}

func (r *Request) FormRequest(method string, uri string, params map[string]string) {
	r.Method = method
	r.Proto = "HTTP/1.1"
	r.Params = params
	if r.Values == nil {
		r.Values = make(map[string]any)
	}
	for key, value := range params {
		r.Values[key] = value
	}
	var host string
	var ok bool
	host, r.Query, ok = strings.Cut(uri, "/")
	if ok {
		r.Query = fmt.Sprintf("/%s", r.Query)
		// fmt.Printf("NewRequest | Query: %s\n", r.Query)
		utils.Debug("Query: %s", r.Query)
	}
	r.Header["Host"] = []string{host}
}

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
	// fmt.Printf("(c *Context) HasEnoughData | length: %d, ReadLength: %d\n", length, r.ReadLength)
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

	// fmt.Printf("(r *Request) ParseFirstLine | Method: %s, Query: %s, Proto: %s\n", r.Method, r.Query, r.Proto)
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

	// fmt.Printf("(r *Request) ParseQuery | Query: %s, params: %s\n", r.Query, params)
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
	// fmt.Printf("(r Request) FormRequest | result: %s\n", string(result))
	utils.Debug("result: %s", string(result))
	return result
}

func (r *Request) Json(obj any) {
	r.Header["Content-Type"] = jsonContentType
	data, _ := json.Marshal(obj)
	r.BodyLength = int32(len(data))
	copy(r.Body[:r.BodyLength], data[:r.BodyLength])
	r.SetContentLength()
}

func (r *Request) Release() {
	r.Method = ""
	r.Query = ""
	r.Proto = ""
	// r.Body = r.Body[:0]
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
	*Context
	Code    int32
	Message string
}

func newResponse(c *Context) *Response {
	r := &Response{
		Context: c,
	}
	return r
}

// Status sets the HTTP response code.
func (r *Response) Status(code int32) {
	r.Code = code
	r.Message = StatusText(code)
}

func (r *Response) Json(code int32, obj any) {
	r.Status(code)

	for k := range r.Context.Header {
		delete(r.Header, k)
	}

	r.Header["Content-Type"] = jsonContentType
	data, _ := json.Marshal(obj)
	r.BodyLength = int32(len(data))
	copy(r.Body[:r.BodyLength], data[:r.BodyLength])
	r.SetContentLength()
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
	// fmt.Printf("(r *Response) ParseFirstLine | Proto: %s, Code: %d, Message: %s\n", r.Proto, r.Code, r.Message)
	utils.Debug("Proto: %s, Code: %d, Message: %s", r.Proto, r.Code, r.Message)
	return true
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
	r.Code = 0
	r.Message = ""
}
