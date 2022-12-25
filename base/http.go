package base

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/pkg/errors"
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
type Header map[string][]string

// A MIMEHeader represents a MIME-style header mapping keys to sets of values.
type MIMEHeader map[string][]string

// ====================================================================================================
// Request & Response
// ====================================================================================================
type R2 struct {
	// 0: 讀取第一行, 1: 讀取 Header, 2: 讀取 Data
	State int8
	*Request
	*Response
	Header

	// 讀取長度
	ReadLength int32

	// Body 數據
	Body []byte
}

func NewR2() *R2 {
	rr := &R2{
		State:    0,
		Request:  NewRequest(),
		Response: NewResponse(),
		Header:   map[string][]string{},
	}
	return rr
}

func (rr *R2) HasLineData(buffer *[]byte, i int32, o int32, length int32) bool {
	fmt.Printf("(rr *R2) HasLineData | i: %d, o: %d, length: %d\n", i, o, length)

	if length == 0 {
		return false
	}

	rr.ReadLength = 0
	value := -1

	if o < i {
		value = bytes.IndexByte((*buffer)[o:i], '\n')
		fmt.Printf("(rr *R2) HasLineData | value: %d\n", value)

	} else {
		value = bytes.IndexByte((*buffer)[o:], '\n')

		if value != -1 {
			rr.ReadLength = int32(value)
			fmt.Printf("(rr *R2) HasLineData | value: %d\n", value)
			return true
		}

		rr.ReadLength = int32(len((*buffer)[o:]))
		fmt.Printf("(rr *R2) HasLineData | temp ReadLength: %d\n", rr.ReadLength)
		value = bytes.IndexByte((*buffer)[:i], '\n')
	}

	if value != -1 {
		rr.ReadLength += int32(value)
		fmt.Printf("(rr *R2) HasLineData | value: %d\n", value)
		return true
	}

	rr.ReadLength = 0
	return false
}

func (rr *R2) HasEnoughData(buffer *[]byte, i int32, o int32, length int32) bool {
	fmt.Printf("(rr *R2) HasEnoughData | length: %d, ReadLength: %d\n", length, rr.ReadLength)
	return length >= rr.ReadLength
}

func (rr *R2) SetHeader(key string, value string) {
	if _, ok := rr.Header[key]; !ok {
		rr.Header[key] = []string{value}
	} else {
		rr.Header[key] = append(rr.Header[key], value)
	}
}

func (rr *R2) SetBody(body []byte) {
	rr.Header["Content-Length"] = []string{strconv.Itoa(len(body))}
	rr.Body = body
}

// TODO: 生成 Response message
func (rr *R2) FormResponse() []byte {
	var buffer bytes.Buffer
	// HTTP/1.1 200 OK\r\n
	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n", rr.Proto, rr.Code, rr.Message))

	// Header
	for k, v := range rr.Header {
		buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, strings.Join(v, ", ")))
	}

	if _, ok := rr.Header["Content-Length"]; ok {
		buffer.WriteString("\r\n")
		buffer.Write(rr.Body)
	}

	return buffer.Bytes()
}

// ====================================================================================================
// Request
// ====================================================================================================
type Request struct {
	Method string
	Query  string
	//
	Proto  string
	Params map[string]string
}

func NewRequest() *Request {
	r := &Request{
		Proto:  "HTTP/1.1",
		Params: map[string]string{},
	}
	return r
}

// 解析第一行數據
// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func (r *Request) ParseFirstLine(line string) bool {
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
	r.Proto = strings.TrimRight(r.Proto, "\r\n")

	fmt.Printf("(r *Request) ParseFirstLine | Method: %s, Query: %s, Proto: %s\n", r.Method, r.Query, r.Proto)
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

	fmt.Printf("(r *Request) ParseQuery | Query: %s, params: %s\n", r.Query, params)
	err := r.ParseParams(params)

	if err != nil {
		return true, errors.Wrapf(err, "Failed to parse params: %s", params)
	}

	fmt.Printf("(r *Request) ParseQuery | params: %+v\n", r.Params)
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
			fmt.Printf("(r *Request) ParseParams | invalid semicolon separator in query(%s)\n", key)
			continue
		}

		if key == "" {
			fmt.Printf("(r *Request) ParseParams | Empty query is found.\n")
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

// ====================================================================================================
// Response
// ====================================================================================================
type Response struct {
	Code    int32
	Message string
}

func NewResponse() *Response {
	r := &Response{}
	return r
}
