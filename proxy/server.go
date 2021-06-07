package proxy

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

type PServer struct {
	rBuf []byte
	net.Conn
}

func NewPServer(conn net.Conn) *PServer {
	return &PServer{
		Conn: conn,
		rBuf: make([]byte, 2),
	}
}

func (p *PServer) Forward() {
	if _, err := io.ReadFull(p.Conn, p.rBuf[:1]); err != nil {
		log.Printf("UNABLE TO GET DST DOMAIN LEN: %v", err)
		p.Conn.Close()
		return
	}

	length := int(p.rBuf[0])
	buf := make([]byte, length + 2)

	if _, err := io.ReadFull(p.Conn, buf); err != nil {
		log.Printf("UNABLE TO GET DST DOMAIN: %v", err)
		p.Conn.Close()
		return
	}

	addr := fmt.Sprintf("%s:%d", string(buf[:length]), binary.BigEndian.Uint16(buf[length:]))
	log.Printf("CONNECTING: %s", addr)
	dst, err := net.DialTimeout("tcp", addr, time.Second * 5)
	if err != nil {
		log.Printf("UNABLE TO CONNECT: %s, %v", addr, err)
		p.Conn.Close()
		return 
	}

	Pipe(p.Conn, dst)
}