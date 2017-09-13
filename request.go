package http1

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"sync"
)

type badStringError struct {
	what string
	str  string
}

func (e *badStringError) Error() string { return fmt.Sprintf("%s %q", e.what, e.str) }

var requestPool sync.Pool

type Request struct {
	Header *RequestHeader
	Body   io.Reader
}

func AcquireRequest() (r *Request) {
	if x := requestPool.Get(); x == nil {
		r = &Request{Header: NewRequestHeader()}
	} else {
		r = x.(*Request)
		r.Header.reset()
		r.resetBody()
	}
	return
}

func ReleaseRequest(r *Request) {
	requestPool.Put(r)
}

func (m *Request) Read(r *bufio.Reader) (err error) {
	m.Header.reset()
	if err = m.Header.Read(r); err == nil {
		m.readBody(r)
	}

	return
}

func (m *Request) resetBody() {
	if m.Body != nil {
		switch r := m.Body.(type) {
		case *chunkedReader:
			releaseChunkedReader(r)
		case *io.LimitedReader:
			releaseLimitedReader(r)
		}

		m.Body = nil
	}
}

func (m *Request) readBody(br *bufio.Reader) {
	m.resetBody()

	if m.Header.GetChunkedEncoding() {
		m.Body = acquireChunkedReader(br)

	} else if contentLength := m.Header.GetContentLength(); contentLength > 0 {
		m.Body = acquireLimitedReader(br, int64(contentLength))

	} else if bytes.Equal(m.Header.Method, bGET) || bytes.Equal(m.Header.Method, bHEAD) {
		m.Body = http.NoBody
	} else {
		// CONNECT / Connection:close / ...
		m.Body = br // read until EOF
	}
}

func (m *Request) WriteTo(w io.Writer) (n int, err error) {
	var written int
	written, err = m.Header.WriteTo(w)
	n += written

	if err == nil && m.Body != nil {
		var written64 int64
		written64, err = io.Copy(w, m.Body)
		n += int(written64)
	}
	return
}

func (m *Request) Method() string {
	return string(m.Header.Method)
}

func (m *Request) RequestURI() string {
	return string(m.Header.RequestURI)
}

var colonSlashSlash = []byte("://")
var bHTTPS = []byte("https")

func (m *Request) GetHostPort() (host string, port int, err error) {
	var bAddr []byte
	ruri := m.Header.RequestURI

	// CONNECT host:port HTTP/1.1
	if bytes.IndexByte(ruri, '/') == -1 && bytes.IndexByte(ruri, ':') != -1 {
		bAddr = ruri
	}

	// GET http://host/ HTTP/1.1
	begin := bytes.Index(ruri, colonSlashSlash)
	if begin > -1 {
		begin += len(colonSlashSlash)
		end := bytes.IndexByte(ruri[begin:], '/')
		if end != -1 {
			end = begin + end
			bAddr = ruri[begin:end]
		}
	}

	// GET / HTTP/1.1\r\nHost: xxx.com\br\n
	if len(bAddr) == 0 {
		bAddr = m.Header.Get(bHost)
	}

	if len(bAddr) == 0 {
		err = fmt.Errorf("missing host port")
	}

	var iColon = bytes.IndexByte(bAddr, ':')
	if iColon == -1 {
		host = string(bAddr)
		if bytes.HasPrefix(ruri, bHTTPS) {
			port = 443
		} else {
			port = 80
		}
	} else {
		host = string(bAddr[:iColon])
		port, _, err = parseUintBuf(bAddr[iColon+1:])
	}

	return
}

func NewRequest(method, urlStr string, body io.Reader) (req *Request) {
	req = AcquireRequest()
	req.Header = new(RequestHeader)
	req.Header.Method = append(req.Header.Method[:0], method...)
	req.Header.RequestURI = append(req.Header.RequestURI[:0], urlStr...)
	req.Header.Proto = append(req.Header.Proto[:0], "HTTP/1.1"...)
	req.Body = body
	return
}

func ReadRequest(r *bufio.Reader) (req *Request, err error) {
	req = AcquireRequest()
	err = req.Read(r)
	return
}
