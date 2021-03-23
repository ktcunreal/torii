package utils

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/crypto/nacl/secretbox"
	"io"
	"net"
	"time"
)

var (
	f = "200601021504"
	M = &MaskParam{}
)

type EncryptedStream struct {
	net.Conn
	rBuf   []byte
	sNonce [24]byte
	rNonce [24]byte
	key    *[32]byte
}

func NewEncryptedStream(conn net.Conn, key *[32]byte) *EncryptedStream {
	return &EncryptedStream{
		key:  key,
		Conn: conn,
	}
}

func (e *EncryptedStream) Read(b []byte) (int, error) {
	if len(e.rBuf) > 0 {
		n := copy(b, e.rBuf)
		e.rBuf = e.rBuf[n:]
		return n, nil
	}

	clb := make([]byte, 8)
	if n, err := io.ReadFull(e.Conn, clb); err != nil {
		return n, err
	}

	cl, ok := Unmask(clb)
	if !ok {
		return 0, nil
	}

	c := make([]byte, cl)
	if n, err := io.ReadFull(e.Conn, c); err != nil {
		return n, err
	}

	p, ok := secretbox.Open([]byte{}, c[:cl], &e.rNonce, e.key)
	increment(&e.rNonce)
	
	if !ok {
		return 0, nil
	}

	n := copy(b, p)
	if n < len(p) {
		e.rBuf = p[n:]
	}

	return n, nil
}

func (e *EncryptedStream) Write(b []byte) (int, error) {
	c := secretbox.Seal([]byte{}, b, &e.sNonce, e.key)
	increment(&e.sNonce)

	clb := Mask(len(c))

	if n, err := e.Conn.Write(clb); err != nil {
		return n, err
	}

	if n, err := e.Conn.Write(c); err != nil {
		return n, err
	}

	return len(b), nil
}

func (e *EncryptedStream) Close() error {
	return e.Conn.Close()
}

func Mask(i int) []byte {
	if !bytes.Equal(M.current_ts, []byte(time.Now().Format(f))) {
		M.Update()
	}

	i_buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(i_buf, uint32(i))

	xor_b := XORBytes(i_buf, M.current_xor)
	return append(M.current_auth, xor_b...)
}

func Unmask(b []byte) (int, bool) {
	xor := make([]byte, 4)

	if !bytes.Equal(M.current_ts, []byte(time.Now().Format(f))) {
		M.Update()
	}

	if bytes.Equal(b[0:4], M.current_auth) {
		xor = M.current_xor
	} else if bytes.Equal(b[0:4], M.lapsed_auth) {
		xor = M.lapsed_xor
	} else {
		return 0, false
	}

	i_buf := XORBytes(b[4:8], xor)
	return int(binary.LittleEndian.Uint32(i_buf)), true
}

func XORBytes(a, b []byte) []byte {
	buf := make([]byte, len(a))
	for i, _ := range a {
		buf[i] = a[i] ^ b[i]
	}
	return buf
}

func increment(b *[24]byte) {
	for i := range b {
		b[i]++
		if b[i] != 0 {
			return
		}
	}
}
