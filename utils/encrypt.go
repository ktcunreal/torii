package utils

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"golang.org/x/crypto/nacl/secretbox"
	"hash"
	"io"
	"net"
)

type EncryptedStream struct {
	net.Conn
	h, m   hash.Hash
	rBuf   []byte
	sNonce [24]byte
	rNonce [24]byte
	key    *[32]byte
}

func NewEncryptedStream(conn net.Conn, key *[32]byte) *EncryptedStream {
	return &EncryptedStream{
		key:  key,
		Conn: conn,
		h:    hmac.New(sha256.New, key[:4]),
		m:    hmac.New(sha256.New, key[28:]),
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

func (e *EncryptedStream) Mask(i int) []byte {
	e.h.Reset()
	e.m.Reset()

	r := make([]byte, 2)
	rand.Read(r[:2])

	e.h.Write(r)
	e.m.Write(r)

	rh := e.h.Sum(nil)[:2]
	mh := e.m.Sum(nil)[:4]

	ib := make([]byte, 4)
	binary.LittleEndian.PutUint32(ib, uint32(i))

	xl := XORBytes(ib, mh)
	rv := append(r, rh...)

	return append(rv, xl...)
}

func (e *EncryptedStream) Unmask(b []byte) (int, bool) {
	e.h.Reset()
	e.m.Reset()

	r := b[:2]

	e.h.Write(r)
	e.m.Write(r)

	rh := e.h.Sum(nil)[:2]
	mh := e.m.Sum(nil)[:4]

	if !hmac.Equal(rh, b[2:4]) {
		return 0, false
	}

	ib := XORBytes(b[4:8], mh)
	i := int(binary.LittleEndian.Uint32(ib))

	return i, true
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
