package utils

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type Socks5Server struct {
	net.Conn
	res, buf  []byte
	ip        net.IP
	port      uint16
	dom, addr string
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
		log.Printf("UNABLE TO GET SOCKS VERSION: %v", err)
		return s.Conn.Close()
	}

	if _, err := s.Conn.Write(s.res[:2]); err != nil {
		log.Printf("UNABLE TO SEND AUTHENTICATION RESPONSE: %v", err)
		return s.Conn.Close()
	}

	if _, err := io.ReadFull(s.Conn, s.buf[:4]); err != nil {
		log.Printf("UNABLE TO READ CLIENT REQUEST: %v", err)
		return s.Conn.Close()
	}

	switch s.buf[3] {
	case 0x01:
		if _, err := io.ReadFull(s.Conn, s.buf[:6]); err != nil {
			log.Printf("UNABLE TO GET DST ADDR: %v", err)
			return s.Conn.Close()
		}
		s.ip, s.port = s.buf[:4], binary.BigEndian.Uint16(s.buf[4:6])
		s.addr = fmt.Sprintf("%s:%d", s.ip, s.port)

	case 0x03:
		if _, err := io.ReadFull(s.Conn, s.buf[:1]); err != nil {
			log.Printf("UNABLE TO GET DST DOMAIN LEN: %v", err)
			return s.Conn.Close()
		}

		l := int(s.buf[0])
		b := make([]byte, l + 2)
		if _, err := io.ReadFull(s.Conn, b); err != nil {
			log.Printf("UNABLE TO GET DST DOMAIN: %v", err)
			return s.Conn.Close()
		}
		s.dom, s.port = string(b[:l]), binary.BigEndian.Uint16(b[l:])
		s.addr = fmt.Sprintf("%s:%d", s.dom, s.port)

	default:
		log.Printf("UNABLE TO PARSE ATYP: %v", s.buf[6])
		return s.Conn.Close()
	}

	dst, err := net.DialTimeout("tcp", s.addr, time.Second*10)
	if err != nil {
		log.Printf("UNABLE TO CONNECT: %s, %v", s.addr, err)
		return s.Conn.Close()
	}

	if _, err := s.Conn.Write(s.res); err != nil {
		log.Printf("UNABLE TO WRITE RESONPSE: %v", err)
		return s.Conn.Close()
	}

	Pipe(s.Conn, dst)
	return nil
}

func Pipe(src, dst net.Conn) {
	go func() {
		defer src.Close()
		defer dst.Close()
		io.Copy(src, dst)
	}()
	go func() {
		defer src.Close()
		defer dst.Close()
		io.Copy(dst, src)
	}()
}
