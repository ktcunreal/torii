package utils

import (
	"github.com/golang/snappy"
	"github.com/klauspost/compress/s2"
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

// S2 wrapper for net.Conn
type S2Stream struct {
	net.Conn
	w *s2.Writer
	r *s2.Reader
}

func NewS2Stream(conn net.Conn) *S2Stream {
	s := &S2Stream{Conn: conn}
	s.w = s2.NewWriter(conn)
	s.r = s2.NewReader(conn)
	return s
}

func (s *S2Stream) Read(p []byte) (n int, err error) {
	return s.r.Read(p)
}

func (s *S2Stream) Write(p []byte) (n int, err error) {
	if _, err := s.w.Write(p); err != nil {
		return 0, err
	}
	if err := s.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), err
}

func (s *S2Stream) Close() error {
	return s.Conn.Close()
}