package http1

import (
	"bufio"
	"bytes"
	"github.com/valyala/fasthttp"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
)

var benchLines = []string{
	"GET / HTTP/1.1",
	"Host: baidu.com",
	"Transfer-encoding: chunked",
	"\r\n",
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
}
var benchBlob = []byte(strings.Join(benchLines, "\r\n"))

var benchR = bytes.NewReader(benchBlob)
var benchBR = bufio.NewReader(benchR)

func resetBR() {
	benchR.Reset(benchBlob)
	benchBR.Reset(benchR)
}

func Benchmark_Overhead(b *testing.B) {
	for i := 0; i < b.N; i++ {
		resetBR()
	}
}

func Benchmark_HttpRequest_Read(b *testing.B) { // 315ns (- overhead), 如果header转化为map会增加400ns
	var req = AcquireRequest()
	for i := 0; i < b.N; i++ {
		resetBR()
		req.Read(benchBR)
	}
	b.ReportAllocs()
}

// fasthttp
// 160ns (- overhead)，其实偷懒没有parse headers
func Benchmark_fasthttp_ReadRequest(b *testing.B) {
	req := fasthttp.AcquireRequest()
	for i := 0; i < b.N; i++ {
		resetBR()
		req.Read(benchBR)
	}
	b.ReportAllocs()
}

func Benchmark_http_ReadRequest(b *testing.B) { // 1620ns (- overhead)
	for i := 0; i < b.N; i++ {
		resetBR()
		http.ReadRequest(benchBR)
	}
	b.ReportAllocs()
}

func Benchmark_RequestHeader_WriteTo(b *testing.B) {
	req := new(Request)
	resetBR()
	req.Read(benchBR)

	for i := 0; i < b.N; i++ {
		req.WriteTo(ioutil.Discard)
	}
	b.ReportAllocs()
}

func Benchmark_fasthttp_ReadRequest_WriteTo(b *testing.B) {
	req := fasthttp.AcquireRequest()
	resetBR()
	req.Read(benchBR)

	for i := 0; i < b.N; i++ {
		req.WriteTo(ioutil.Discard)
	}
	b.ReportAllocs()
}
