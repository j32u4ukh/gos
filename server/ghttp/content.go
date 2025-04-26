package ghttp

import (
	"encoding/json"
	"strconv"

	"github.com/pkg/errors"
)

type Content struct {
	// ex: HTTP/1.1
	httpProto string
	header    Header
	// body 數據
	body       []byte
	bodyLength int32
	// 工作流程當前階段
	state ContextState
}

func newContent() *Content {
	return &Content{
		httpProto:  DEFAULT_HTTP_PROTO,
		header:     make(Header),
		body:       make([]byte, 64*1024),
		bodyLength: 0,
	}
}

func (c *Content) SetHttpProto(httpProto string) {
	c.httpProto = httpProto
}

func (c *Content) SetHeader(key string, value string) {
	c.header[key] = append(c.header[key], value)
}

func (c *Content) setDefaultHeader(key string, defaultValue string) {
	if _, ok := c.header[key]; !ok {
		c.header[key] = []string{defaultValue}
	}
}

// 設置 Body 數據
func (c *Content) SetBody(data []byte) {
	c.bodyLength = int32(len(data))
	c.header[HEADER_CONTENT_LENGTH] = []string{strconv.Itoa(int(c.bodyLength))}
	if int32(cap(c.body)) < c.bodyLength {
		c.body = make([]byte, c.bodyLength)
		copy(c.body, data)
	} else {
		copy(c.body[:c.bodyLength], data)
	}
}

func (c *Content) SetJson(obj any) error {
	c.header[HEADER_CONTENT_TYPE] = []string{CONTENT_TYPE_JSON}
	data, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return errors.Wrap(err, "Failed to marshal object data")
	}
	c.SetBody(data)
	return nil
}

func (c *Content) GetHeader(key string) ([]string, bool) {
	if header, ok := c.header[key]; ok {
		return header, true
	}
	return nil, false
}

func (c *Content) Release() {
	c.httpProto = DEFAULT_HTTP_PROTO
	for k := range c.header {
		delete(c.header, k)
	}
	c.bodyLength = 0
	c.state = READ_FIRST_LINE
}
