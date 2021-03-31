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

type EncryptedStream struct {
	net.Conn
	rBuf   []byte
	sNonce [24]byte
	rNonce [24]byte
	Key    *[32]byte
}

func NewEncryptedStream(conn net.Conn, key *[32]byte) *EncryptedStream {
	return &EncryptedStream{
		Key:  key,
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

	cl, ok := Unmask(clb, (*e.Key)[:])
	if !ok {
		log.Println("INVALID CIPHER LENGTH")
		return e.Drop()
	}

	c := make([]byte, cl)
	if n, err := io.ReadFull(e.Conn, c); err != nil {
		return n, err
	}

	p, ok := secretbox.Open([]byte{}, c[:cl], &e.rNonce, e.Key)
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
	c := secretbox.Seal([]byte{}, b, &e.sNonce, e.Key)
	increment(&e.sNonce)
	clb := Mask(len(c), (*e.Key)[:])

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

func Mask(i int, key []byte) []byte {
	k1 := key[:4]
	k2 := key[28:]

	r := make([]byte, 2)
	rand.Read(r)

	ib := make([]byte, 4)
	binary.LittleEndian.PutUint32(ib, uint32(i))

	sb := make([]byte, 2)
	copy(sb, r)
	am := SH256S(append(sb, k1...))

	nc := make([]byte, 2)
	copy(nc, r)
	rm := SH256S(append(nc, k2...))

	head := append(r, am[2:4]...)
	tail := XORBytes(ib, rm)

	return append(head, tail...)
}

func Unmask(b []byte, key []byte) (int, bool) {
	k1 := key[:4]
	k2 := key[28:]

	sb := make([]byte, 2)
	copy(sb, b[:2])
	am := SH256S(append(sb, k1...))

	if !bytes.Equal(am[2:4], b[2:4]) {
		return 0, false
	}

	nc := make([]byte, 2)
	copy(nc, b[:2])
	rm := SH256S(append(nc, k2...))

	ib := XORBytes(b[4:8], rm)
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

func SH256S(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[28:]
}
