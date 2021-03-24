package utils

import (
	"encoding/binary"
	"golang.org/x/crypto/nacl/secretbox"
	"io"
	"crypto/hmac"
    "crypto/sha256"
	"net"
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
	if _, err := io.ReadFull(e.Conn, clb); err != nil {
		return 0, err
	}

	cl, ok := e.Unmask(clb)
	if !ok {
		return 0, nil
	}

	c := make([]byte, cl)
	if _, err := io.ReadFull(e.Conn, c); err != nil {
		return 0, err
	}

	p, ok := secretbox.Open([]byte{}, c, &e.rNonce, e.key)
	if !ok {
		return 0, nil
	}
	increment(&e.rNonce)

	n := copy(b, p)
	if n < len(p) {
		e.rBuf = p[n:]
	}

	return n, nil
}

func (e *EncryptedStream) Write(b []byte) (int, error) {
	c := secretbox.Seal([]byte{}, b, &e.sNonce, e.key)
	increment(&e.sNonce)
	clb := e.Mask(len(c))

	n, err := e.Conn.Write(append(clb, c...))
	if err != nil {
		return 0, err
	}

	return n, nil
}

func (e *EncryptedStream) Close() error {
	return e.Conn.Close()
}


func (e *EncryptedStream) Mask(i int) []byte{
	kb, ke := (*e.key)[:4], (*e.key)[28:]

	i_buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(i_buf, uint32(i))
	xi_buf := XORBytes(i_buf, kb)

	h := hmac.New(sha256.New, ke)
	h.Write(i_buf)
	hs256 := h.Sum(nil)[:4]

	return append(xi_buf, hs256...)
}

func (e* EncryptedStream) Unmask(b []byte) (int, bool) {
	kb, ke := (*e.key)[:4], (*e.key)[28:]

	xi_buf := b[:4]
	i_buf := XORBytes(xi_buf, kb)
	i := int(binary.LittleEndian.Uint32(i_buf))

	h := hmac.New(sha256.New, ke)
	h.Write(i_buf)
	hs256 := h.Sum(nil)[:4]

	return i, hmac.Equal(hs256, b[4:8])
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

