package ghttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// 工作完成時的 Callback 函式
type HandlerFunc func(*Context)

type Context struct {
	// Context 唯一碼
	id int32
	// 對應 連線物件/工作物件 的 id
	Index int32

	//////////////////////////////////////////////////
	// 0: 讀取第一行, 1: 讀取 Header, 2: 讀取 Data, 3: 等待數據寫出(Response) 4. 完成數據複製到寫出緩存
	State int8
	Header
	*Request2
	*Response2

	// 讀取長度
	ReadLength int32

	// Body 數據
	Body       []byte
	BodyLength int32
}

func NewContext(id int32) *Context {
	c := &Context{
		id:         id,
		Index:      -1,
		State:      0,
		Header:     map[string][]string{},
		Body:       make([]byte, 64*1024),
		BodyLength: 0,
	}
	c.Request2 = newRequest2(c)
	c.Response2 = newResponse2(c)
	return c
}

func (c *Context) SetHeader(key string, value string) {
	if _, ok := c.Header[key]; !ok {
		c.Header[key] = []string{value}
	} else {
		c.Header[key] = append(c.Header[key], value)
	}
}

func (c *Context) setBodyLength() {
	c.Header["Content-Length"] = []string{strconv.Itoa(len(c.Body))}
}

// ====================================================================================================
// Request
// ====================================================================================================
type Request2 struct {
	*Context
	Method string
	Query  string
	//
	Proto  string
	Params map[string]string
}

func newRequest2(c *Context) *Request2 {
	r := &Request2{
		Context: c,
		Proto:   "HTTP/1.1",
		Params:  map[string]string{},
	}
	return r
}

func (r *Request2) HasLineData(buffer *[]byte, i int32, o int32, length int32) bool {
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

func (r *Request2) HasEnoughData(buffer *[]byte, i int32, o int32, length int32) bool {
	fmt.Printf("(c *Context) HasEnoughData | length: %d, ReadLength: %d\n", length, r.ReadLength)
	return length >= r.ReadLength
}

// 解析第一行數據
// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func (r *Request2) ParseFirstLine(line string) bool {
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

	fmt.Printf("(r *Request2) ParseFirstLine | Method: %s, Query: %s, Proto: %s\n", r.Method, r.Query, r.Proto)
	return true
}

// 解析第一行數據中的請求路徑
func (r *Request2) ParseQuery() (bool, error) {
	var ok bool
	var params string
	r.Query, params, ok = strings.Cut(r.Query, "?")

	if !ok {
		return false, nil
	}

	fmt.Printf("(r *Request2) ParseQuery | Query: %s, params: %s\n", r.Query, params)
	err := r.ParseParams(params)

	if err != nil {
		return true, errors.Wrapf(err, "Failed to parse params: %s", params)
	}

	fmt.Printf("(r *Request2) ParseQuery | params: %+v\n", r.Params)
	return true, nil
}

// 解析第一行數據中的請求路徑中的 GET 參數
func (r *Request2) ParseParams(params string) error {
	var key, value string
	var ok bool
	var err error

	for params != "" {
		key, params, _ = strings.Cut(params, "&")

		if strings.Contains(key, ";") {
			fmt.Printf("(r *Request2) ParseParams | invalid semicolon separator in query(%s)\n", key)
			continue
		}

		if key == "" {
			fmt.Printf("(r *Request2) ParseParams | Empty query is found.\n")
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

func (r Request2) GetParam(key string) (bool, string) {
	if param, ok := r.Params[key]; ok {
		return true, param
	}
	return false, ""
}

func (r Request2) ToRequestData() []byte {
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
		buffer.Write(r.Body)
	}
	result := buffer.Bytes()
	fmt.Printf("(r Request2) FormRequest2 | result: %s\n", string(result))
	return result
}

func (r *Request2) Json(obj any) {
	r.Header["Content-Type"] = jsonContentType
	r.Body, _ = json.Marshal(obj)
	r.setBodyLength()
}

// ====================================================================================================
// Response
// ====================================================================================================
type Response2 struct {
	*Context
	Code    int32
	Message string
}

func newResponse2(c *Context) *Response2 {
	r := &Response2{
		Context: c,
	}
	return r
}

// Status sets the HTTP response code.
func (r *Response2) Status(code int32) {
	r.Code = code
	r.Message = StatusText(code)
}

func (r *Response2) Json(code int32, obj any) {
	r.Status(code)

	for k := range r.Context.Header {
		delete(r.Header, k)
	}

	r.Header["Content-Type"] = jsonContentType
	r.Body, _ = json.Marshal(obj)
	r.setBodyLength()
}

// 解析第一行數據
// parseRequestLine parses "HTTP/1.1 200 OK" into its three parts.
func (r *Response2) ParseFirstLine(line string) bool {
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
	fmt.Printf("(r *Response2) ParseFirstLine | Proto: %s, Code: %d, Message: %s\n", r.Proto, r.Code, r.Message)
	return true
}

// 生成 Response message
func (r Response2) ToResponseData() []byte {
	var buffer bytes.Buffer
	// HTTP/1.1 200 OK\r\n
	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n", r.Proto, r.Code, r.Message))

	// Header
	for k, v := range r.Header {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}

	if _, ok := r.Header["Content-Length"]; ok {
		buffer.WriteString("\r\n")
		buffer.Write(r.Body)
	}

	return buffer.Bytes()
}
