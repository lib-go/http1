package http1

import (
	"bufio"
	"bytes"
	"errors"
	"io"
	"sync"
)

var (
	ErrInvalidChunkHeaderLength = errors.New("invalid chunk header length")
	ErrInvalidChunkHeader       = errors.New("invalid chunk header")
	ErrInvalidChunkEnding       = errors.New("invalid chunk ending")
	ErrChunkLengthTooLarge      = errors.New("chunk length too large")
)

var limitedReaderPool sync.Pool

func acquireLimitedReader(r io.Reader, n int64) io.Reader {
	if lr := limitedReaderPool.Get(); lr == nil {
		return io.LimitReader(r, n)
	} else {
		olr := lr.(*io.LimitedReader)
		olr.R = r
		olr.N = n
		return olr
	}
}

func releaseLimitedReader(lr *io.LimitedReader) {
	lr.R = nil
	limitedReaderPool.Put(lr)
}

var chunkedReaderPool sync.Pool

func acquireChunkedReader(br *bufio.Reader) *chunkedReader {
	cr := chunkedReaderPool.Get()
	if cr == nil {
		return &chunkedReader{br: br}
	} else {
		return cr.(*chunkedReader).Reset(br)
	}
}

func releaseChunkedReader(r *chunkedReader) {
	r.Reset(nil)
	chunkedReaderPool.Put(r)
}

type chunkedReader struct {
	br  *bufio.Reader
	err error
	n   uint64 // unread chunk body bytes
	end bool   // 写完body的n字节就结束
}

func (cr *chunkedReader) Reset(br *bufio.Reader) (*chunkedReader) {
	cr.br = br
	cr.err = nil
	cr.n = 0
	cr.end = false
	return cr
}

func (cr *chunkedReader) beginChunk() {
	var b []byte
	if b, cr.err = peekUntil(cr.br, CRLF); cr.err != nil {
		return
	}

	if cr.n, cr.err = parseChunkHeaderLength(b); cr.err == nil {
		if cr.n == 0 {
			cr.end = true
		}
		cr.n += uint64(len(b) + 2) // chunk head and tailing CRLF
	}
}

func (cr *chunkedReader) Read(b []uint8) (n int, err error) {
	var n0 int

	for cr.err == nil {
		if len(b) == 0 {
			break
		}

		// 写chunk
		if cr.n > 0 {
			b0 := b
			if uint64(len(b0)) > cr.n {
				b0 = b0[:cr.n]
			}
			n0, cr.err = cr.br.Read(b0)
			n += n0
			b = b[n0:]
			cr.n -= uint64(n0)

			// 检查结尾合法性，是否为\r\n
			if cr.n == 0 && n0 > 2 && !bytes.Equal(b0[len(b0)-2:], CRLF) {
				cr.err = ErrInvalidChunkEnding
			}
			continue
		}

		if cr.end {
			cr.err = io.EOF
			break
		}

		if cr.n == 0 {
			cr.beginChunk()
			continue
		}
	}

	if cr.err == io.EOF && (!cr.end || cr.n != 0) {
		cr.err = io.ErrUnexpectedEOF
	}

	return n, cr.err
}

func parseChunkHeaderLength(v []byte) (n uint64, err error) {
	for i, b := range v {
		switch {
		case '0' <= b && b <= '9':
			b = b - '0'
		case 'a' <= b && b <= 'f':
			b = b - 'a' + 10
		case 'A' <= b && b <= 'F':
			b = b - 'A' + 10
		case b == ';' || b == '\r' || b == '\n':
			// 数字区合法结束，停止读取
			if i == 0 {
				// 如果开头就是非数字，则报错
				return 0, ErrInvalidChunkHeaderLength
			}
			return
		default:
			return 0, ErrInvalidChunkHeader
		}
		if i == 16 {
			return 0, ErrChunkLengthTooLarge
		}
		n <<= 4
		n |= uint64(b)
	}
	return
}
