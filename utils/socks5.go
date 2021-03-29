package utils

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"time"
)

type Socks5Server struct {
	net.Conn
	res, buf []byte
	addr     string
	ip       net.IP
}

func NewSocks5Server(conn net.Conn) *Socks5Server {
	return &Socks5Server{
		res:  []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		buf:  make([]byte, 8),
		Conn: conn,
	}
}

func (s *Socks5Server) Proxy() error {
	if _, err := io.ReadFull(s.Conn, s.buf[:3]); err != nil {
		return s.Conn.Close()
	}

	if _, err := s.Conn.Write(s.res[:2]); err != nil {
		return s.Conn.Close()
	}

	if _, err := io.ReadFull(s.Conn, s.buf[:4]); err != nil {
		return s.Conn.Close()
	}

	switch s.buf[3] {

	case 0x03:
		if _, err := io.ReadFull(s.Conn, s.buf[:1]); err != nil {
			return s.Conn.Close()
		}

		dom_buf := make([]byte, int(s.buf[0]))

		if _, err := io.ReadFull(s.Conn, dom_buf); err != nil {
			return s.Conn.Close()
		}

		if _, err := io.ReadFull(s.Conn, s.buf[6:]); err != nil {
			return s.Conn.Close()
		}

		s.addr = fmt.Sprintf("%s:%d", string(dom_buf), binary.BigEndian.Uint16(s.buf[6:]))

	default:
		return s.Conn.Close()
	}

	dst, err := net.DialTimeout("tcp", s.addr, time.Second*5)
	if err != nil {
		return s.Conn.Close()
	}

	if _, err := s.Conn.Write(s.res); err != nil {
		return s.Conn.Close()
	}

	go Pipe(s.Conn, dst)
	return nil
}

func Pipe(src, dst net.Conn) {
	copyConn := func(a, b net.Conn) {
		defer a.Close()
		defer b.Close()
		io.Copy(a, b)
		return
	}
	go copyConn(src, dst)
	go copyConn(dst, src)
}
