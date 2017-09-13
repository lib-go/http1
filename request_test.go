package http1

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"net"
	"testing"
	"strings"
	"bytes"
	"time"
)

func Test_Request_Read(t *testing.T) {
	chunkedRequest := []byte(strings.Join([]string{
		"POST / HTTP/1.1",
		"Host: baidu.com",
		"Transfer-Encoding: chunked",
		"",
		"4",
		"Wiki",
		"5",
		"pedia",
		"E",
		" in",
		"",
		"chunks.",
		"0",
		"\r\n",
	}, "\r\n"))

	left, right := net.Pipe()
	br := bufio.NewReader(right)
	go left.Write(chunkedRequest)

	req, e := ReadRequest(br)
	assert.Nil(t, e)
	assert.Equal(t, "POST", req.Method())
	assert.Equal(t, "/", req.RequestURI())

	w := bytes.NewBuffer(nil)
	req.WriteTo(w)
	assert.Equal(t, chunkedRequest, w.Bytes())
	ReleaseRequest(req)

	limitedRequest := []byte(strings.Join([]string{
		"POST / HTTP/1.1",
		"Content-Length: 5",
		"",
		"12345",
	}, "\r\n"))
	go left.Write(limitedRequest)
	req = AcquireRequest()
	e = req.Read(br)
	assert.Nil(t, e)
	w = bytes.NewBuffer(nil)
	req.WriteTo(w)
	assert.Equal(t, limitedRequest, w.Bytes())
	ReleaseRequest(req)

	readUntilCloseRequest := []byte(strings.Join([]string{
		"POST / HTTP/1.1",
		"",
		"123456",
	}, "\r\n"))

	go left.Write(readUntilCloseRequest)
	req = AcquireRequest()
	time.AfterFunc(time.Millisecond*100, func() {
		left.Close()
	})
	e = req.Read(br)
	assert.Nil(t, e)
	w = bytes.NewBuffer(nil)
	req.WriteTo(w)
	assert.Equal(t, readUntilCloseRequest, w.Bytes())

}

func Test_GetHostPort(t *testing.T) {
	req := NewRequest("GET", "http://baidu.com/", nil)
	req.Header.Add([]byte("Host"), []byte("baidu.com"))

	host, port, e := req.GetHostPort()
	assert.Equal(t, "baidu.com", host)
	assert.Equal(t, 80, port)
	assert.Nil(t, e)

	req = NewRequest("GET", "/", nil)
	req.Header.Add([]byte("Host"), []byte("baidu.com"))

	host, port, e = req.GetHostPort()
	assert.Equal(t, "baidu.com", host)
	assert.Equal(t, 80, port)
	assert.Nil(t, e)

	req = NewRequest("CONNECT", "google.com:443", nil)
	req.Header.Add([]byte("Host"), []byte("google"))

	host, port, e = req.GetHostPort()
	assert.Equal(t, "google.com", host)
	assert.Equal(t, 443, port)
	assert.Nil(t, e)
}
