package http1

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/stretchr/testify/assert"
	"strings"
	"testing"
)

func readRequestHeader(lines []string) (h *RequestHeader, e error) {
	br := bufio.NewReader(bytes.NewReader([]byte(strings.Join(lines, "\r\n"))))
	h = NewRequestHeader()
	e = h.Read(br)
	return
}

func Test_RequestHeader_Read(t *testing.T) {
	var h *RequestHeader
	var e error

	h, e = readRequestHeader([]string{
		"CONNECT google.com:443 HTTP/1.1",
		"\r\n",
	})
	assert.Nil(t, e)
	assert.Equal(t, []byte("CONNECT"), h.Method)
	assert.Equal(t, []byte("google.com:443"), h.RequestURI)
	assert.Equal(t, []byte("HTTP/1.1"), h.Proto)
	assert.Equal(t, -1, h.GetContentLength())
	assert.False(t, h.GetChunkedEncoding())

	h, e = readRequestHeader([]string{
		"CONNECT google.com:443",
		"\r\n",
	})
	assert.Equal(t, []byte(""), h.Proto)

	h, e = readRequestHeader([]string{
		"GET / HTTP/1.1",
		"Host: baidu.com",
		"Connection: close",
		"Content-Length: 128",
		"Transfer-Encoding: gzip; chunked;",
		"\r\n",
	})
	assert.Nil(t, e)
	assert.Equal(t, []byte("GET"), h.Method)
	assert.Equal(t, []byte("/"), h.RequestURI)
	assert.Equal(t, []byte("HTTP/1.1"), h.Proto)
	assert.Equal(t, []byte("baidu.com"), h.Get(bHost))
	assert.Equal(t, []byte("close"), h.Get(bConnection))
	assert.Equal(t, 128, h.GetContentLength())
	assert.True(t, h.GetChunkedEncoding())
}

func Test_RequestHeaderGetAddDel(t *testing.T) {
	h, e := readRequestHeader([]string{
		"GET / HTTP/1.1",
		"Host: baidu.com",
		"Connection: close",
		"Content-Length: 128",
		"Transfer-Encoding: chunked",
		"\r\n",
	})
	assert.Nil(t, e)

	// Get
	assert.Equal(t, []byte("baidu.com"), h.Get(bHost))

	// GetChunkedEncoding
	assert.True(t, h.GetChunkedEncoding())
	h.Add(bTransferEncoding, []byte("deflate"))
	assert.True(t, h.GetChunkedEncoding())
	h.Del(bTransferEncoding)
	assert.False(t, h.GetChunkedEncoding())

	// Add
	h.Add([]byte("Content-Type"), []byte("application/json"))
	assert.Equal(t, []byte("application/json"), h.Get([]byte("Content-Type")))

	// Del
	deleted := h.Del([]byte("Hello-World"))
	assert.Equal(t, 0, deleted, "删除不存在的header")

	h.Add([]byte("Content-Type"), []byte("text/html"))
	deleted = h.Del([]byte("Content-Type"))
	assert.Equal(t, 2, deleted, "可删除多个header")
	assert.Nil(t, h.Get([]byte("Content-Type")))

	// Set
	modified := h.Set(bHost, []byte("163.com"))
	assert.Equal(t, 1, modified)
	assert.Equal(t, []byte("163.com"), h.Get(bHost))

	h.Add([]byte("Host"), []byte("google.com"))
	modified = h.Set(bHost, []byte("iplocation.net"))
	assert.Equal(t, 2, modified)
	assert.Equal(t, []byte("iplocation.net"), h.Get(bHost))

	// 检查内容是否正确
	b := h.Bytes()
	fmt.Println(string(b))
}

func Test_splitHeaders(t *testing.T) {
	lines := []string{
		"GET / HTTP/1.1",
		"Host: baidu.com",
		"Connection: close",
		"\r\n",
	}
	buf := []byte(strings.Join(lines, "\r\n"))

	headers := make([][]byte, 10)
	headers = splitHeaders(headers, buf)

	assert.Equal(t, 3, len(headers))
	assert.Equal(t, []byte(lines[0]), headers[0])
	assert.Equal(t, []byte(lines[1]), headers[1])

	fmt.Println(headers)
}

func Test_readUntil(t *testing.T) {
	lines := []string{
		"Host: baidu.com",
		"Connection: close",
		"\r\n",
	}
	buf := []byte(strings.Join(lines, "\r\n"))

	r := bufio.NewReader(bytes.NewBuffer(buf))
	ret, e := peekUntil(r, []byte("\r\n\r\n"))
	assert.Equal(t, buf, ret)
	assert.Nil(t, e)
}

func Test_parseHeaderKey(t *testing.T) {
	key, value := splitHeaderKeyValue([]byte("Host: baidu.com"))
	assert.Equal(t, []byte("Host"), key)
	assert.Equal(t, []byte("baidu.com"), value)

	key, value = splitHeaderKeyValue([]byte("Host   :    baidu.com"))
	assert.Equal(t, []byte("Host"), key)
	assert.Equal(t, []byte("baidu.com"), value)
}
