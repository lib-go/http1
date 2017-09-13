package http1

import (
	"bufio"
	"bytes"
	"io"
	"net"
	"net/http"
)

func IsWebRequest(req *Request) bool {
	return req.Method() != http.MethodConnect && !bytes.Contains(req.Header.RequestURI, []byte("://"))
}

type restoredConn struct {
	net.Conn
	r io.Reader
}

func (c *restoredConn) Read(b []byte) (n int, e error) {
	return c.r.Read(b)
}

func RestoreConn(conn net.Conn, br *bufio.Reader, req *Request) net.Conn {
	var readers []io.Reader

	if req != nil {
		readers = append(readers, bytes.NewBuffer(req.Header.Bytes()))
	}
	if br != nil && br.Buffered() > 0 {
		readers = append(readers, io.LimitReader(br, int64(br.Buffered())))
	}
	readers = append(readers, conn)

	return &restoredConn{
		Conn: conn,
		r:    io.MultiReader(readers...),
	}
}
