package ghttp

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"

	"github.com/j32u4ukh/gos/log"
	"github.com/pkg/errors"
)

type Request struct {
	*Content
	// ex: GET
	method string
	// ex: http/https
	scheme string
	// ex: /user/get
	query string
	// params 參數
	params map[string]string
	values map[string]any
}

func NewRequest() *Request {
	r := &Request{
		Content: newContent(),
		params:  make(map[string]string),
		values:  make(map[string]any),
	}
	r.state = READ_FIRST_LINE
	return r
}

func (r *Request) SetRequest(method string, uri string, query map[string]string) error {
	r.method = method
	// 解析 URI
	u, err := url.Parse(uri)
	if err != nil {
		return errors.Wrapf(err, "Uri(%s) 解析失敗", uri)
	}
	r.scheme = u.Scheme
	r.header["Host"] = []string{u.Host}
	// path 預設為 "/"，避免空字串
	r.query = u.Path
	if r.query == "" {
		r.query = "/"
	}
	// 確保 params 不為 nil
	if query == nil {
		query = make(map[string]string)
	}
	// 將原始 URI 中的 query 參數合併進 params（可被覆蓋）
	queryValues := u.Query()
	for k, v := range queryValues {
		if len(v) > 0 {
			query[k] = v[0]
		}
	}
	// 實際構建 query string
	if len(query) > 0 {
		values := url.Values{}
		for k, v := range query {
			values.Set(k, v)
		}
		r.query += "?" + values.Encode()
		r.params = query
	}
	return nil
}

// 解析第一行數據
// parseRequestLine parses "GET /foo HTTP/1.1" into its three parts.
func (r *Request) parseFirstLine(line string) bool {
	var ok bool
	r.method, r.query, ok = strings.Cut(line, " ")
	if !ok {
		return false
	}
	r.query, r.httpProto, ok = strings.Cut(r.query, " ")
	if !ok {
		return false
	}
	r.query = strings.TrimPrefix(r.query, "?")
	return true
}

// 解析第一行數據中的請求路徑
func (r *Request) parseQuery() (bool, error) {
	var ok bool
	var params string
	r.query, params, ok = strings.Cut(r.query, "?")
	if !ok {
		return false, nil
	}
	log.Debug("Query: %s, params: %s", r.query, params)
	err := r.parseParams(params)
	if err != nil {
		return true, errors.Wrapf(err, "Failed to parse params: %s", params)
	}
	log.Debug("params: %+v", r.params)
	return true, nil
}

// 解析第一行數據中的請求路徑中的 GET 參數
func (r *Request) parseParams(params string) error {
	var key, value string
	var ok bool
	var err error
	for params != "" {
		key, params, _ = strings.Cut(params, "&")
		if strings.Contains(key, ";") {
			log.Warn("invalid semicolon separator in query(%s)", key)
			continue
		}
		if key == "" {
			log.Warn("Empty query is found.")
			continue
		}
		key, value, ok = strings.Cut(key, "=")
		if !ok {
			continue
		}
		// 將 url 上的參數加入 params 管理
		r.params[key] = value
	}
	return err
}

func (r Request) Method() string {
	return r.method
}

func (r Request) Query() string {
	return r.query
}

func (r Request) GetParam(key string) (string, bool) {
	if param, ok := r.params[key]; ok {
		return param, true
	}
	return "", false
}

func (r Request) GetValue(key string) (any, bool) {
	if value, ok := r.values[key]; ok {
		return value, true
	}
	return "", false
}

//  Accept: */*
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
func (r *Request) Bytes() []byte {
	var buffer bytes.Buffer
	// 請求行: GET /path?query HTTP/1.1
	buffer.WriteString(fmt.Sprintf("%s %s %s\r\n", r.method, r.query, r.httpProto))
	// 預設標頭（如果未手動設置）
	r.setDefaultHeader(HEADER_USER_AGENT, "Go-http-client/1.1")
	r.setDefaultHeader(HEADER_ACCEPT_ENCODING, "gzip")
	r.setDefaultHeader(HEADER_CONNECTION, "close")
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

func (r *Request) Release() {
	r.method = ""
	r.query = ""
	r.httpProto = DEFAULT_HTTP_PROTO
	r.bodyLength = 0
	var key string
	for key = range r.params {
		delete(r.params, key)
	}
	for key = range r.header {
		delete(r.header, key)
	}
}

func (r Request) String() string {
	return string(r.Bytes())
}
