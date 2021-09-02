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

const Chunk int = 16384

type Key struct {
	hash   *[32]byte
	salt   []byte
	pepper []byte
}

func NewKey(s string) *Key {
	h := SH256R(s)
	return &Key{
		hash:   &h,
		salt:   SH256L(h[:10]),
		pepper: SH256L(h[20:]),
	}
}

type EncStream struct {
	net.Conn
	key            *Key
	rBuf, dBuf     []byte
	rNonce, sNonce [24]byte
}

func NewEncStream(conn net.Conn, psk *Key) *EncStream {
	return &EncStream{
		Conn: conn,
		key:  psk,
		dBuf: make([]byte, 12),
	}
}

func (e *EncStream) Read(b []byte) (int, error) {
	if len(e.rBuf) > 0 {
		n := copy(b, e.rBuf)
		e.rBuf = e.rBuf[n:]
		return n, nil
	}

	if n, err := io.ReadFull(e.Conn, e.dBuf); err != nil || n != 12 {
		return 0, err
	}

	size, ok := Decode(e.dBuf, e.key.salt, e.key.pepper)
	if !ok {
		log.Println("INVALID CIPHER LENGTH")
		return e.Drop()
	}

	c := make([]byte, size)
	if _, err := io.ReadFull(e.Conn, c); err != nil {
		log.Println("ERROR READING CIPHER")
		return 0, err
	}

	p, ok := secretbox.Open(nil, c[:size], &e.rNonce, e.key.hash)
	if !ok {
		log.Println("DECRYPT FAILED")
		return e.Drop()
	}
	increment(&e.rNonce)

	n := copy(b, p)
	if n < len(p) {
		e.rBuf = p[n:]
	}

	return n, nil
}

func (e *EncStream) Write(b []byte) (int, error) {
	sidx, eidx := 0, 0
	for ; sidx < len(b); sidx = eidx {
		if len(b)-eidx >= Chunk {
			eidx += Chunk
		} else {
			eidx = len(b)
		}
		cipher := secretbox.Seal([]byte{}, b[sidx:eidx], &e.sNonce, e.key.hash)
		increment(&e.sNonce)
		enc := Encode(len(cipher), e.key.salt, e.key.pepper)
		if _, err := e.Conn.Write(enc); err != nil {
			return sidx, err
		}
		if _, err := e.Conn.Write(cipher); err != nil {
			return sidx, err
		}
	}
	return sidx, nil
}

func (e *EncStream) Close() error {
	return e.Conn.Close()
}

func (e *EncStream) Drop() (int, error) {
	defer e.Conn.Close()
	trap := make([]byte, 12)
	for {
		_, err := io.ReadFull(e.Conn, trap)
		if err != nil {
			break
		}
	}
	return 0, errors.New("ILLEGAL CONNECTION")
}

func Encode(i int, s, p []byte) []byte {
	r := make([]byte, 4)
	rand.Read(r)

	enc := make([]byte, 4)
	binary.LittleEndian.PutUint32(enc, uint32(i))

	tmp := make([]byte, 4)
	copy(tmp, r)
	auth := SH256S(append(tmp, s...))

	tmp = make([]byte, 4)
	copy(tmp, r)
	mask := SH256S(append(tmp, p...))

	head := append(r, auth...)
	xorenc := XORBytes(enc, mask)

	return append(head, xorenc...)
}

func Decode(b []byte, s, p []byte) (int, bool) {
	tmp := make([]byte, 4)
	copy(tmp, b[:4])
	auth := SH256S(append(tmp, s...))

	if !bytes.Equal(auth, b[4:8]) {
		return 0, false
	}

	tmp = make([]byte, 4)
	copy(tmp, b[:4])
	mask := SH256S(append(tmp, p...))

	enc := XORBytes(b[8:12], mask)
	i := int(binary.LittleEndian.Uint32(enc))

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

func SH256L(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

func SH256S(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[28:]
}

func SH256R(s string) [32]byte {
	return sha256.Sum256([]byte(s))
}
