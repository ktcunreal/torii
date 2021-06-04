package utils

import (
	"github.com/golang/snappy"
	"github.com/valyala/gozstd"
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

// zstd wrapper for net.Conn
type ZstdStream struct {
	net.Conn
	r *gozstd.Reader
	w *gozstd.Writer
}

func NewZstdStream(conn net.Conn) *ZstdStream {
	c := &ZstdStream{Conn: conn}
	c.r = gozstd.NewReader(conn)
	c.w = gozstd.NewWriter(conn)
	return c
}

func (z *ZstdStream) Read(p []byte) (n int, err error) {
	return z.r.Read(p)
}

func (z *ZstdStream) Write(p []byte) (n int, err error) {
	if _, err := z.w.Write(p); err != nil {
		return 0, err
	}
	if err := z.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), err
}

func (z *ZstdStream) Close() error {
	return z.Conn.Close()
}