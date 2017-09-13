package http1

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
)

var (
	CRLF     = []byte("\r\n")
	CRLFCRLF = []byte("\r\n\r\n")

	bTransferEncoding = []byte("Transfer-Encoding")
	bContentLength    = []byte("Content-Length")
	bConnection       = []byte("Connection")

	bChunked = []byte("chunked")
	bGET     = []byte("GET")
	bHEAD    = []byte("HEAD")
	bHost    = []byte("Host")
)

type RequestHeader struct {
	Method     []byte
	RequestURI []byte
	Proto      []byte

	headers [][]byte
}

func NewRequestHeader() (h *RequestHeader) {
	h = new(RequestHeader)
	// 预分配一个合适的大小
	h.Method = make([]byte, 0, 8)
	h.RequestURI = make([]byte, 0, 16)
	h.Proto = make([]byte, 0, 8)
	h.headers = make([][]byte, 0, 5)

	return
}

func (h *RequestHeader) Read(r *bufio.Reader) (err error) {

	var b []byte
	if b, err = peekUntil(r, CRLF); err != nil {
		if err == io.EOF && len(b) > 0 {
			err = io.ErrUnexpectedEOF
		}
		return
	}
	mustDiscard(r, len(b))

	h.parseFirstLine(b[:len(b)-2])

	// 检查是否firstLine之后就结束了（0个http头）
	if b, err = r.Peek(2); err != nil {
		if err == io.EOF && len(b) != 2 {
			err = io.ErrUnexpectedEOF
		}
		return
	}
	if bytes.Equal(b, CRLF) {
		mustDiscard(r, len(b))
		h.headers = make([][]byte, 3)
		return
	}

	// 有1个以上http头的情况
	if b, err = peekUntil(r, CRLFCRLF); err != nil {
		if err == io.EOF && !bytes.HasSuffix(b, CRLFCRLF) {
			err = io.ErrUnexpectedEOF
		}
		return
	}
	mustDiscard(r, len(b))

	h.headers = splitHeaders(h.headers, b)
	for _, header := range h.headers {
		if len(header) > 0 {
			normalizeHeaderKey(header)
		}
	}
	return
}

func (h *RequestHeader) reset() {
	h.Method = h.Method[:0]
	h.RequestURI = h.RequestURI[:0]
	h.Proto = h.Proto[:0]
	h.headers = h.headers[:0]
}

func (h *RequestHeader) parseFirstLine(b []byte) error {
	// parse Method
	n := bytes.IndexByte(b, ' ')
	if n <= 0 {
		return fmt.Errorf("cannot find http request Method in %q", b)
	}
	h.Method = append(h.Method[:0], b[:n]...)
	b = b[n+1:]

	// parse RequestURI
	n = bytes.LastIndexByte(b, ' ')
	if n < 0 {
		h.Proto = h.Proto[:0]
		n = len(b)
	} else if n == 0 {
		return fmt.Errorf("RequestURI cannot be empty in %q", b)
	} else {
		h.Proto = append(h.Proto[:0], b[n+1:]...)
	}
	h.RequestURI = append(h.RequestURI[:0], b[:n]...)

	return nil
}

func (h *RequestHeader) VisitFor(key []byte, f func(i int, value []byte) bool) {
	l := len(key)
	if l == 0 {
		return
	}
	for i, header := range h.headers {
		if len(header) > l && (header[l] == ' ' || header[l] == ':') && bytes.Equal(header[:l], key) {
			value := header[l+1:]
			for skip := 0; skip < len(value); skip++ {
				if value[skip] != ' ' {
					value = value[skip:]
					break
				}
			}

			if !f(i, value) {
				return
			}
		}
	}
}

func (h *RequestHeader) Get(key []byte) (value []byte) {
	h.VisitFor(key, func(i int, v []byte) bool {
		value = v
		return false
	})
	return
}

func (h *RequestHeader) GetChunkedEncoding() (yes bool) { // 32ns
	h.VisitFor(bTransferEncoding, func(i int, value []byte) bool {
		if bytes.Contains(value, bChunked) {
			yes = true
			return false
		}
		return true
	})

	return
}

func (h *RequestHeader) GetContentLength() (n int) {
	n = -1
	h.VisitFor(bContentLength, func(i int, value []byte) bool {
		n, _, _ = parseUintBuf(value)
		return false
	})

	return
}

func (h *RequestHeader) Add(key, value []byte) {
	if len(key) == 0 {
		return
	}
	newHeader := append(key, ':', ' ')
	newHeader = append(newHeader, value...)
	h.headers = append(h.headers, newHeader)
}

func (h *RequestHeader) Del(key []byte) (n int) {
	h.VisitFor(key, func(i int, value []byte) bool {
		n += 1
		h.headers[i] = h.headers[i][:0]
		return true
	})

	return
}

func (h *RequestHeader) Set(key, value []byte) (n int) {
	h.VisitFor(key, func(i int, v []byte) bool {
		h.headers[i] = h.headers[i][:0]

		// 仅修改第一个遇到的，其他删除
		if n == 0 {
			h.headers[i] = append(key, ':', ' ')
			h.headers[i] = append(h.headers[i], value...)
		}

		n += 1
		return true
	})

	if n == 0 {
		h.Add(key, value)
	}

	return
}

func (h *RequestHeader) Bytes() []byte {
	// 先计算b的大小，再分配，减少后续append过程中的内存申请次数（计算大小几乎不耗时间）
	sz := len(h.Method) + len(h.RequestURI) + len(h.Proto) + 4
	for _, header := range h.headers {
		if len(header) > 0 {
			sz += len(header) + 2
		}
	}
	sz += 2
	b := make([]byte, 0, sz)

	// 首行
	b = append(b, h.Method...)
	b = append(b, ' ')
	b = append(b, h.RequestURI...)
	b = append(b, ' ')
	b = append(b, h.Proto...)
	b = append(b, CRLF...)

	// headers
	for _, header := range h.headers {
		if len(header) > 0 {
			b = append(b, header...)
			b = append(b, CRLF...)
		}
	}

	b = append(b, CRLF...)

	return b
}

func (h *RequestHeader) WriteTo(w io.Writer) (n int, e error) {
	return w.Write(h.Bytes())
}

func splitHeaders(headers [][]byte, buf []byte) [][]byte {
	/*
		benchmark：
		- 直接bytes.Split 				250ns
		- headers=make([][]byte, p1) 	120ns
		- 外界提供 headers 				30ns

		所以还是用第三种方式
	*/

	headers = headers[:0]

	p1 := 0
	p2 := 0
	for {
		p2 = bytes.IndexByte(buf[p1:], '\r')
		if p2 > 0 {
			headers = append(headers, buf[p1: p1+p2])
		}
		p1 += p2 + 2
		if p1 >= len(buf) {
			break
		}
	}

	return headers
}

func mustPeekBuffered(r *bufio.Reader) []byte {
	buf, err := r.Peek(r.Buffered())
	if len(buf) == 0 || err != nil {
		panic(fmt.Sprintf("bufio.Reader.Peek() returned unexpected data (%q, %v)", buf, err))
	}
	return buf
}

func mustDiscard(r *bufio.Reader, n int) {
	if _, err := r.Discard(n); err != nil {
		panic(fmt.Sprintf("bufio.Reader.Discard(%d) failed: %s", n, err))
	}
}

func peekUntil(r *bufio.Reader, sep []byte) (b []byte, err error) {
	n := 1
	for {
		if b, err = r.Peek(n); err != nil {
			return
		}
		b = mustPeekBuffered(r)

		i := bytes.Index(b, sep)
		if i != -1 {
			b = b[:i+len(sep)]
			break
		} else {
			n += 1
		}
	}
	return
}

func normalizeHeaderKey(b []byte) {
	n := len(b)
	if n == 0 {
		return
	}

	b[0] = toUpperTable[b[0]]
	for i := 1; i < n; i++ {
		p := &b[i]
		if *p == ':' {
			break
		}
		if *p == '-' {
			i++
			if i < n {
				b[i] = toUpperTable[b[i]]
			}
			continue
		}

		*p = toLowerTable[*p]
	}
}

func splitHeaderKeyValue(header []byte) (key, value []byte) {
	m := bytes.IndexByte(header, ':')
	for i := m; i > 1; i-- {
		if header[i-1] != ' ' {
			key = header[:i]
			break
		}
	}

	for i := m + 1; i < len(header); i++ {
		if header[i] != ' ' {
			value = header[i:]
			break
		}
	}
	return
}
