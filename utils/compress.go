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

func (l *LZ4Stream) Close() error {
	return l.Conn.Close()
}