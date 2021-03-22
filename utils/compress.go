package utils

import (
	"github.com/golang/snappy"
	"net"
)

type CompStream struct {
	net.Conn
	w *snappy.Writer
	r *snappy.Reader
}

func (c *CompStream) Read(p []byte) (n int, err error) {
	return c.r.Read(p)
}

func (c *CompStream) Write(p []byte) (n int, err error) {

	if _, err := c.w.Write(p); err != nil {
		return 0, err
	}

	if err := c.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), err
}

func (c *CompStream) Close() error {
	return c.Conn.Close()
}

func NewCompStream(conn net.Conn) *CompStream {
	c := &CompStream{Conn: conn}
	c.w = snappy.NewBufferedWriter(conn)
	c.r = snappy.NewReader(conn)
	return c
}
