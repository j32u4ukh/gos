package ghttp

import (
	"bytes"
	"strconv"
	"strings"

	"github.com/j32u4ukh/gos/base"
	"github.com/j32u4ukh/gos/log"
	"github.com/pkg/errors"
)

/*
Request 和 Response 的 Header 應該分開來，因為端點函式中既會讀取 Request 的 Header，也會寫出 Response 的 Header。
透過不同函式來讀寫即可，記得保留 HttpContext 送出 Request 的能力。
*/
type HttpContext struct {
	*base.Context
	req *Request
	res *Response
	// 數據模式
	mode HttpMode
	// 工作流程當前階段
	state ContextState
}

func NewHttpContext() *HttpContext {
	c := &HttpContext{
		Context: base.NewContext(nil),
		req:     NewRequest(),
		res:     newResponse(),
		mode:    HTTPMODE_REQUEST,
		state:   READ_FIRST_LINE,
	}
	return c
}

func (c *HttpContext) setHttpMode(mode HttpMode) {
	c.mode = mode
}

func (c *HttpContext) Read() error {
	var err error
	data := c.Conn.GetData()
	isReadable := true
	for isReadable {
		switch c.state {
		case READ_FIRST_LINE:
			data, isReadable = c.readFirstLine(data)
		case READ_HEADER:
			data, isReadable, err = c.readHeader(data)
			if err != nil {
				return errors.Wrap(err, "An error occurred while reading the header.")
			}
		case READ_BODY:
			c.readBody(data)
			isReadable = false
		}
		if c.state == WRITE_RESPONSE || c.state == READ_RESPONSE_FINISH {
			break
		}
	}
	return nil
}

func (c *HttpContext) readFirstLine(data []byte) ([]byte, bool) {
	index := bytes.IndexByte(data, '\n')
	if index != -1 {
		length := int32(index) + 1
		// 拆分第一行數據
		bs, err := c.GetBuffer().FetchByteArray(uint32(length))
		if err != nil {
			return data, false
		}
		line := strings.TrimRight(string(bs), "\r\n")
		data = data[length:]
		switch c.mode {
		case HTTPMODE_REQUEST:
			c.req.parseFirstLine(line)
			if c.req.Method() == METHOD_GET {
				// 解析第一行數據中的請求路徑
				c.req.parseQuery()
			}
		case HTTPMODE_RESPONSE:
			c.res.parseFirstLine(line)
		}
		// 切換到讀取 Header 階段
		c.state = READ_HEADER
		return data, true
	}
	return data, false
}

func (c *HttpContext) readHeader(data []byte) ([]byte, bool, error) {
	var content *Content = c.getContent()
	var key, value, header string
	var ok bool
	var index, length int
	for c.state == READ_HEADER {
		index = bytes.IndexByte(data, '\n')
		if index == -1 {
			break
		}
		length = index + 1
		bs, err := c.GetBuffer().FetchByteArray(uint32(length))
		if err != nil {
			return data, false, errors.Wrap(err, "Failed to read header")
		}
		header = strings.TrimRight(string(bs), "\r\n")
		header = strings.TrimSpace(header)
		data = data[length:]
		if header == "" {
			// Header 結束，檢查是否有 Content-Length 決定下一步行為
			if contentLength, ok := content.header[HEADER_CONTENT_LENGTH]; ok {
				if len(contentLength) == 0 {
					return data, false, errors.New("Content-Length header present but empty")
				}
				length, err := strconv.Atoi(contentLength[0])
				if err != nil {
					return data, false, errors.Wrap(err, "Failed to read Content-Length")
				}
				content.bodyLength = int32(length)
				// 切換到讀取 Body 階段
				c.state = READ_BODY
				log.Debug("State: READ_HEADER -> READ_BODY(length: %d)", length)
				return data, len(data) >= length, nil
			} else {
				// 無 Body，進入下一階段：等待寫入或接收回應
				switch c.mode {
				case HTTPMODE_REQUEST:
					// 切換到等待寫入 Response
					c.state = WRITE_RESPONSE
				case HTTPMODE_RESPONSE:
					// 切換到讀取回應 Response
					c.state = READ_RESPONSE_FINISH
				}
				log.Debug("State: READ_HEADER -> %s", c.state)
				return data, true, nil
			}
		} else if key, value, ok = strings.Cut(header, COLON); ok {
			// mustHaveFieldNameColon ensures that, per RFC 7230, the field-name is on a single line,
			// so the first line must contain a colon.
			// 將讀到的數據從冒號拆分成 key, value
			value = strings.TrimSpace(value)
			content.SetHeader(key, value)
		} else {
			return data, false, errors.Errorf("Invalid header line (no colon): %s", header)
		}
	}
	return data, false, nil
}

func (c *HttpContext) readBody(data []byte) {
	var content *Content = c.getContent()
	if int32(len(data)) < content.bodyLength {
		// 資料不足，等待下一個封包
		return
	}
	bs, err := c.GetBuffer().FetchByteArray(uint32(content.bodyLength))
	if err != nil {
		return
	}
	// 將傳入的數據，加入工作緩存中
	content.SetBody(bs)
	// 等待數據寫出
	switch c.mode {
	case HTTPMODE_REQUEST:
		// Body 讀完，準備寫出 Response
		c.state = WRITE_RESPONSE
	case HTTPMODE_RESPONSE:
		// Body 讀完，進入等待下一次處理階段
		c.state = READ_RESPONSE_FINISH
	}
	log.Debug("State: READ_BODY -> %s", c.state)
}

func (c *HttpContext) getContent() *Content {
	switch c.mode {
	case HTTPMODE_REQUEST:
		return c.req.Content
	case HTTPMODE_RESPONSE:
		return c.res.Content
	default:
		return nil
	}
}

func (c *HttpContext) SetHeader(key string, value string) {
	c.res.SetHeader(key, value)
}

func (c *HttpContext) Status(code int32) {
	c.res.Status(code)
}

func (c *HttpContext) Json(code int32, obj any) error {
	c.res.Status(code)
	err := c.res.SetJson(obj)
	if err != nil {
		return errors.Wrap(err, "")
	}
	return nil
}

func (c *HttpContext) Release() {
	c.mode = HTTPMODE_REQUEST
	c.state = READ_FIRST_LINE
	c.Context.Release()
	c.req.Release()
	c.res.Release()
}
