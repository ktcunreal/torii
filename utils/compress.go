package utils

import (
	"github.com/golang/snappy"
	"net"
)

// snappy wrapper for net.Conn
type SnappyStream struct {
	net.Conn
	w *snappy.Writer
	r *snappy.Reader
}

func NewSnappyStream(conn net.Conn) *SnappyStream {
	s := &SnappyStream{Conn: conn}
	s.w = snappy.NewBufferedWriter(conn)
	s.r = snappy.NewReader(conn)
	return s
}

func (s *SnappyStream) Read(p []byte) (n int, err error) {
	return s.r.Read(p)
}

func (s *SnappyStream) Write(p []byte) (n int, err error) {
	if _, err := s.w.Write(p); err != nil {
		return 0, err
	}

	if err := s.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), err
}

func (s *SnappyStream) Close() error {
	return s.Conn.Close()
}
