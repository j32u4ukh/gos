package base

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// A Header represents the key-value pairs in an HTTP header.
//
// The keys should be in canonical form, as returned by
// CanonicalHeaderKey.
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

	// 讀取長度
	ReadLength int32
}

func NewR2() *R2 {
	rr := &R2{
		State:    0,
		Request:  NewRequest(),
		Response: NewResponse(),
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

// ====================================================================================================
// Request
// ====================================================================================================
type Request struct {
	Method string
	Query  string
	Proto  string
	Params map[string]string
	Header
	Data []byte
}

func NewRequest() *Request {
	r := &Request{
		Header: map[string][]string{},
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
}

func NewResponse() *Response {
	r := &Response{}
	return r
}
