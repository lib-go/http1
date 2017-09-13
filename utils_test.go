package http1

import (
	"bufio"
	"fmt"
	"github.com/stretchr/testify/assert"
	"net"
	"net/http"
	"sync"
	"testing"
)

func TestRestoreConn(t *testing.T) {
	ln, _ := net.Listen("tcp4", "127.0.0.1:0")

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		conn, e := ln.Accept()
		assert.Nil(t, e)

		br := bufio.NewReader(conn)
		req, e := ReadRequest(br)
		assert.Nil(t, e)

		conn2 := RestoreConn(conn, br, req)
		br2 := bufio.NewReader(conn2)
		req2, e := ReadRequest(br2)

		assert.Equal(t, req.Header.Bytes(), req2.Header.Bytes(), "从原始conn中读到的req和恢复后的conn中读到的req一样")
		wg.Done()
	}()

	go http.Get(fmt.Sprintf("http://%s/", ln.Addr()))

	wg.Wait()
}
