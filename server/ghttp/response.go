package ghttp

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/j32u4ukh/gos/log"
)

type Response struct {
	*Content
	code    int32
	message string
}

func newResponse() *Response {
	r := &Response{
		Content: newContent(),
		code:    -1,
		message: "",
	}
	return r
}

// 解析第一行數據
// parseRequestLine parses "HTTP/1.1 200 OK" into its three parts.
func (r *Response) parseFirstLine(line string) bool {
	var ok bool
	r.httpProto, r.message, ok = strings.Cut(line, " ")
	if !ok {
		return false
	}
	var codeString string
	codeString, r.message, ok = strings.Cut(r.message, " ")
	if !ok {
		r.message = codeString
		return false
	}
	code, err := strconv.Atoi(codeString)
	if err != nil {
		return false
	}
	r.code = int32(code)
	log.Debug("Proto: %s, Code: %d, Message: %s", r.httpProto, r.code, r.message)
	return true
}

// Status sets the HTTP response code.
func (r *Response) Status(code int32) {
	r.code = code
	r.message = StatusText(code)
}

// 生成 Response message
func (r Response) Bytes() []byte {
	var buffer bytes.Buffer
	// HTTP/1.1 200 OK\r\n
	buffer.WriteString(fmt.Sprintf("%s %d %s\r\n", r.httpProto, r.code, r.message))
	// 寫入所有標頭
	for k, vlist := range r.header {
		for _, v := range vlist {
			buffer.WriteString(fmt.Sprintf("%s: %s\r\n", k, v))
		}
	}
	// 空行分隔 header 和 body
	buffer.WriteString("\r\n")
	// 如果有 body，就寫入 body
	if r.bodyLength > 0 {
		buffer.Write(r.body[:r.bodyLength])
	}
	return buffer.Bytes()
}

func (r Response) String() string {
	return string(r.Bytes())
}

func (r *Response) Release() {
	r.code = -1
	r.message = ""
	r.bodyLength = 0
	r.body = r.body[:0]
	for k := range r.header {
		delete(r.header, k)
	}
}
