package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type ProxyServer struct {
	rBuf []byte
	net.Conn
}

func NewProxyServer(conn net.Conn) *ProxyServer {
	return &ProxyServer{
		Conn: conn,
		rBuf: make([]byte, 2),
	}
}

func (p *ProxyServer) Connect() {
	if _, err := io.ReadFull(p.Conn, p.rBuf[:1]); err != nil {
		log.Printf("%v", err)
		defer p.Conn.Close()
		return
	}

	length := int(p.rBuf[0])
	buf := make([]byte, length+2)

	if n, err := io.ReadFull(p.Conn, buf); err != nil || n != length+2 {
		log.Printf("ILLEGAL DST: %v", err)
		defer p.Conn.Close()
		return
	}

	addr := fmt.Sprintf("%s:%d", string(buf[:length]), binary.BigEndian.Uint16(buf[length:]))
	log.Printf("CONNECTING: %s", addr)

	dst, err := net.DialTimeout("tcp", addr, time.Second*15)
	if err != nil {
		log.Printf("UNABLE TO CONNECT: %s, %v", addr, err)
		defer p.Conn.Close()
		return
	}

	Pipe(p.Conn, dst)
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
