package http1

import (
	"testing"
	"fmt"
	"strings"
	"bytes"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"bufio"
)

func Test_parseChunkHeaderLength(t *testing.T) {
	var (
		s []byte
		n uint64
		e error
	)

	s = []byte("0")
	n, e = parseChunkHeaderLength(s)
	assert.Equal(t, uint64(0), n)
	assert.Nil(t, e)

	s = []byte("24")
	n, e = parseChunkHeaderLength(s)
	assert.Equal(t, uint64(0x24), n)
	assert.Nil(t, e)

	s = []byte("F3;  ")
	n, e = parseChunkHeaderLength(s)
	assert.Equal(t, uint64(0xF3), n)
	assert.Nil(t, e)

	s = []byte("24***\r\n")
	n, e = parseChunkHeaderLength(s)
	assert.NotNil(t, e)

	s = []byte("\r\n")
	n, e = parseChunkHeaderLength(s)
	assert.NotNil(t, e)

	_ = n
}

func TestChunkedReader_Examples(t *testing.T) {
	var (
		okChunk string = "5\r\nhello\r\n0\r\n\r\n"
		s       string
		e       error
	)

	readChunkString := func(s string) (string, error) {
		cr := acquireChunkedReader(bufio.NewReader(bytes.NewReader([]byte(s))))
		b, e := ioutil.ReadAll(cr)
		releaseChunkedReader(cr)
		return string(b), e
	}

	s, e = readChunkString(okChunk)
	assert.Equal(t, okChunk, s)

	s, e = readChunkString("5\r\nhello\r\n\r\n0\r\n\r\n")
	assert.Equal(t, ErrInvalidChunkHeaderLength, e)

	s, e = readChunkString("5\r\nhello**0\r\n\r\n")
	assert.Equal(t, ErrInvalidChunkEnding, e)

	s, e = readChunkString("5\r\nhello\r\n0\r\n")
	fmt.Println(s)
}

func Test_ChunkedReader_Read_DifferentSize(t *testing.T) {
	lines := []string{
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
	blob := []byte(strings.Join(lines, "\r\n"))

	cr := acquireChunkedReader(bufio.NewReader(bytes.NewReader(blob)))
	for bsize := 100; bsize >= 1; bsize -- {
		var result []byte
		var e error
		var n int

		// 用不同尺寸都能读取完整
		b := make([]byte, bsize)
		for {
			if n, e = cr.Read(b); n > 0 {
				result = append(result, b[:n]...)
				fmt.Printf(".")
			} else {
				break
			}
		}
		//fmt.Println(result)
		assert.Equal(t, blob, result, fmt.Sprintf("%v", bsize))
		fmt.Println(bsize, e)

		cr.Reset(bufio.NewReader(bytes.NewReader(blob)))
	}
}
