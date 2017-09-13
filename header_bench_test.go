package http1

import (
	"testing"
	"strings"
	"bufio"
	"bytes"
)

func Benchmark_splitHeaders(b *testing.B) {
	buf := []byte(strings.Join([]string{
		"Host: baidu.com",
		"Connection: close",
		"\r\n",
	}, "\r\n"))

	headers := make([][]byte, 10)
	headers = splitHeaders(headers, buf)

	for i := 0; i < b.N; i++ {
		splitHeaders(headers, buf)
	}
}

func Benchmark_peekUntil(b *testing.B) {
	lines := []string{
		"Host: baidu.com",
		"Connection: close",
		"\r\n",
	}
	buf := []byte(strings.Join(lines, "\r\n"))
	r := bufio.NewReader(bytes.NewBuffer(buf))

	for i := 0; i < b.N; i++ {
		peekUntil(r, []byte("\r\n\r\n"))
	}
}

func Benchmark_RequestHeader_Get(b *testing.B) {
	h := new(RequestHeader)
	h.headers = [][]byte{
		[]byte("Host: baidu.com"),               // 16ns
		[]byte("Content-Type:application/json"), // 18ns
		[]byte("User-Agent:   go"),              // 21ns
		[]byte("Content-Length: 100"),           // 22ns
		nil,
		[]byte("Transfer-Encoding: chunked"), // 25ns
	}

	m := map[string]string{
		"Host":              "baidu.com",
		"Content-Type":      "application/json",
		"User-Agent":        "go",
		"Content-Length":    "100",
		"Transfer-Encoding": "chunked", // 13.3ns
	}
	_ = m["Transfer-Encoding"]

	for i := 0; i < b.N; i++ {
		//h.Get([]byte("Host"))
		//h.Get([]byte("Content-Type"))
		//h.Get([]byte("User-Agent"))
		//h.Get([]byte("Content-Length"))
		h.Get(bTransferEncoding)
		//h.GetChunkedEncoding()
		//h.GetContentLength()

		//_ = m["Host"]
		//_ = m["Content-Type"]
		//_ = m["User-Agent"]
		//_ = m["Transfer-Encoding"]
	}
	b.ReportAllocs()
}

func Benchmark_RequestHeader_Add(b *testing.B) { // 44ns
	h := new(RequestHeader)

	h.headers = [][]byte{
		[]byte("Host: baidu.com"),
		[]byte("Content-Type:application/json"),
		[]byte("User-Agent:   go"),
		[]byte("Content-Length: 100"),
		[]byte("Transfer-Encoding: chunked"),
	}

	m := map[string]string{
		"Host":              "baidu.com",
		"Content-Type":      "application/json",
		"User-Agent":        "go",
		"Content-Length":    "100",
		"Transfer-Encoding": "chunked",
	}
	_ = m["Transfer-Encoding"]

	k := []byte("Connection")
	v := []byte("close")
	for i := 0; i < b.N; i++ {
		h.Add(k, v)
		h.headers = h.headers[:5]
	}
}

func Benchmark_RequestHeader_Set(b *testing.B) { // 37ns
	h := new(RequestHeader)

	h.headers = [][]byte{
		[]byte("Host: baidu.com"),
		[]byte("Content-Type:application/json"),
		[]byte("User-Agent:   go"),
		[]byte("Content-Length: 100"),
		[]byte("Transfer-Encoding: chunked"),
		nil,
		nil,
		nil,
	}

	k := []byte("Transfer-Encoding")
	v := []byte("gzip")
	for i := 0; i < b.N; i++ {
		h.Set(k, v)
	}
	b.ReportAllocs()
}

func Benchmark_RequestHeader_Bytes(b *testing.B) { // 74ns
	lines := []string{
		"GET / HTTP/1.1",
		"Host: baidu.com",
		"Connection: close",
		"\r\n",
	}
	buf := []byte(strings.Join(lines, "\r\n"))
	r := bufio.NewReader(bytes.NewBuffer(buf))

	h := new(RequestHeader)
	h.Read(r)

	for i := 0; i < b.N; i++ {
		h.Bytes()
	}
}
