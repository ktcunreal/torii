package proxy

import (
	"io"
	"log"
	"net"
)

var res = []byte{0x05, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

type ProxyClient struct {
	rBuf []byte
	net.Conn
}

func NewProxyClient(conn net.Conn) *ProxyClient {
	return &ProxyClient{
		Conn: conn,
		rBuf: make([]byte, 4),
	}
}

func (p *ProxyClient) Connect(src net.Conn) {
	if _, err := io.ReadFull(p.Conn, p.rBuf[:3]); err != nil {
		log.Printf("UNABLE TO GET SOCKS VERSION: %v", err)
		p.Conn.Close()
		return
	}

	if _, err := p.Conn.Write(res[:2]); err != nil {
		log.Printf("UNABLE TO SEND RESPONSE: %v", err)
		p.Conn.Close()
		return
	}

	if _, err := io.ReadFull(p.Conn, p.rBuf[:4]); err != nil {
		log.Printf("UNABLE TO READ CLIENT REQUEST: %v", err)
		p.Conn.Close()
		return
	}

	if p.rBuf[3] != 0x03 {
		log.Printf("UNSUPPORTED ATYP")
		p.Conn.Close()
		return
	}

	if _, err := p.Conn.Write(res); err != nil {
		log.Printf("UNABLE TO WRITE RESPONSE: %v", err)
		p.Conn.Close()
		return
	}
	Pipe(p.Conn, src)
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
