package utils

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"golang.org/x/crypto/nacl/secretbox"
	"io"
	"log"
	"net"
)

var H1, H2 []byte

type EncryptedStream struct {
	net.Conn
	rBuf   []byte
	sNonce [24]byte
	rNonce [24]byte
	key    *[32]byte
}

func NewEncryptedStream(conn net.Conn, k *[32]byte) *EncryptedStream {
	return &EncryptedStream{
		key:  k,
		Conn: conn,
	}
}

func (e *EncryptedStream) Read(b []byte) (int, error) {
	if len(e.rBuf) > 0 {
		n := copy(b, e.rBuf)
		e.rBuf = e.rBuf[n:]
		return n, nil
	}

	clb := make([]byte, 12)
	if n, err := io.ReadFull(e.Conn, clb); err != nil {
		return n, err
	}

	cl, ok := Unmask(clb)
	if !ok {
		log.Println("INVALID CIPHER LENGTH")
		return e.Drop()
	}

	c := make([]byte, cl)
	if n, err := io.ReadFull(e.Conn, c); err != nil {
		return n, err
	}

	p, ok := secretbox.Open([]byte{}, c[:cl], &e.rNonce, e.key)
	increment(&e.rNonce)
	if !ok {
		log.Println("DECRYPT FAILED")
		return e.Drop()
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

func (e *EncryptedStream) Drop() (int, error) {
	defer e.Conn.Close()
	d := make([]byte, 16)
	for {
		_, err := io.ReadFull(e.Conn, d)
		if err != nil {
			break
		}
	}
	return 0, errors.New("ILLEGAL CONNECTION ABORTED")
}

func Mask(i int) []byte {
	r := make([]byte, 4)
	rand.Read(r)

	ib := make([]byte, 4)
	binary.LittleEndian.PutUint32(ib, uint32(i))

	sb := make([]byte, 4)
	copy(sb, r)
	am := SH256S(append(sb, H1...))

	nc := make([]byte, 4)
	copy(nc, r)
	rm := SH256S(append(nc, H2...))

	header := append(r, am...)
	l := XORBytes(ib, rm)

	return append(header, l...)
}

func Unmask(b []byte) (int, bool) {
	sb := make([]byte, 4)
	copy(sb, b[:4])
	am := SH256S(append(sb, H1...))

	if !bytes.Equal(am, b[4:8]) {
		return 0, false
	}

	nc := make([]byte, 4)
	copy(nc, b[:4])
	rm := SH256S(append(nc, H2...))

	ib := XORBytes(b[8:12], rm)
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

func SH256(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

func SH256S(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[28:]
}
