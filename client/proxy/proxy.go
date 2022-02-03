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
	if n, err := io.ReadFull(p.Conn, p.rBuf[:3]); err != nil || n != 3 {
		log.Printf("UNABLE TO GET SOCKS VERSION: %v", err)
		defer p.Conn.Close()
		return
	}

	if n, err := p.Conn.Write(res[:2]); err != nil || n != 2 {
		log.Printf("UNABLE TO SEND RESPONSE: %v", err)
		defer p.Conn.Close()
		return
	}

	if n, err := io.ReadFull(p.Conn, p.rBuf[:4]); err != nil || n != 4 {
		log.Printf("UNABLE TO GET CLIENT REQUEST: %v", err)
		defer p.Conn.Close()
		return
	}

	if p.rBuf[3] != 0x03 {
		log.Printf("UNSUPPORTED ATYP")
		defer p.Conn.Close()
		return
	}

	if n, err := p.Conn.Write(res); err != nil || n != 10 {
		log.Printf("UNABLE TO WRITE RESPONSE: %v", err)
		defer p.Conn.Close()
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
