package utils

import (
	"github.com/golang/snappy"
	"github.com/pierrec/lz4"
	"net"
)

// snappy wrapper for net.Conn
type SnappyStream struct {
	net.Conn
	w *snappy.Writer
	r *snappy.Reader
}

func NewSnappyStream(conn net.Conn) *SnappyStream {
	c := &SnappyStream{Conn: conn}
	c.w = snappy.NewBufferedWriter(conn)
	c.r = snappy.NewReader(conn)
	return c
}

func (c *SnappyStream) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *SnappyStream) Write(p []byte) (n int, err error) {
	if _, err := c.w.Write(p); err != nil {
		return 0, err
	}

	if err := c.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), err
}

func (c *SnappyStream) Close() error {
	return c.Conn.Close()
}

// LZ4 wrapper for net.Conn
type LZ4Stream struct {
	net.Conn
	w *lz4.Writer
	r *lz4.Reader
}

func NewLZ4Stream(conn net.Conn) *LZ4Stream {
	l := &LZ4Stream{Conn: conn}
	l.w = lz4.NewWriter(conn)
	l.r = lz4.NewReader(conn)
	return l
}

func (l *LZ4Stream) Read(p []byte) (n int, err error) {
	return l.r.Read(p)
}

func (l *LZ4Stream) Write(p []byte) (n int, err error) {
	if _, err := l.w.Write(p); err != nil {
		return 0, err
	}
	if err := l.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), err
}