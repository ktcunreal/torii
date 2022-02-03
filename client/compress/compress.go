package compress

import (
	"github.com/andybalholm/brotli"
	"github.com/golang/snappy"
	"net"
)

type SnappyStreamClient struct {
	net.Conn
	r *snappy.Reader
}

func NewSnappyStreamClient(conn net.Conn) *SnappyStreamClient {
	s := &SnappyStreamClient{Conn: conn}
	s.r = snappy.NewReader(conn)
	return s
}

func (s *SnappyStreamClient) Read(b []byte) (n int, err error) {
	return s.r.Read(b)
}

func (s *SnappyStreamClient) Write(b []byte) (n int, err error) {
	return s.Conn.Write(b)
}

func (s *SnappyStreamClient) Close() error {
	return s.Conn.Close()
}

// Duplex snappy wrapper
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

func (s *SnappyStream) Read(b []byte) (n int, err error) {
	return s.r.Read(b)
}

func (s *SnappyStream) Write(b []byte) (n int, err error) {
	if _, err := s.w.Write(b); err != nil {
		return 0, err
	}
	if err := s.w.Flush(); err != nil {
		return 0, err
	}
	return len(b), err
}

func (s *SnappyStream) Close() error {
	return s.Conn.Close()
}

// Duplex brotli wrapper
type BrotliStream struct {
	net.Conn
	w *brotli.Writer
	r *brotli.Reader
}

func NewBrotliStream(conn net.Conn) *BrotliStream {
	b := &BrotliStream{Conn: conn}
	b.w = brotli.NewWriter(conn)
	b.r = brotli.NewReader(conn)
	return b
}

func (b *BrotliStream) Read(p []byte) (n int, err error) {
	return b.r.Read(p)
}

func (b *BrotliStream) Write(p []byte) (n int, err error) {
	if _, err := b.w.Write(p); err != nil {
		return 0, err
	}
	if err := b.w.Flush(); err != nil {
		return 0, err
	}
	return len(p), err
}

func (b *BrotliStream) Close() error {
	return b.Conn.Close()
}
