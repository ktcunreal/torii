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

var (
	HKEY1, HKEY2 []byte
	Chunk int = 1024 * 16
)

type EncStream struct {
	net.Conn
	rBuf, dBuf []byte
	sNonce [24]byte
	rNonce [24]byte
	psk    *[32]byte
}

func NewEncStream(c net.Conn, k *[32]byte) *EncStream {
	return &EncStream{
		Conn: c,
		psk:  k,
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

	size, ok := Decode(e.dBuf)
	if !ok {
		log.Println("INVALID CIPHER LENGTH")
		return e.Drop()
	}

	c := make([]byte, size)
	if _, err := io.ReadFull(e.Conn, c); err != nil {
		log.Println("ERROR READING CIPHER")
		return 0, err
	}

	p, ok := secretbox.Open(nil, c[:size], &e.rNonce, e.psk)
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
		if len(b) - eidx >= Chunk {
			eidx += Chunk
		} else {
			eidx = len(b)
		}
			
		cipher := secretbox.Seal([]byte{}, b[sidx:eidx], &e.sNonce, e.psk)

		wBuf := make([]byte, len(cipher) + 12)
		eBuf := Encode(len(cipher))
	
		copy(wBuf[:12], eBuf)
		copy(wBuf[12:], cipher)
	
		if _, err := e.Conn.Write(wBuf); err != nil {
			return sidx, err
		}

		increment(&e.sNonce)
	}
	return sidx, nil	
}

func (e *EncStream) Close() error {
	return e.Conn.Close()
}

func (e *EncStream) Drop() (int, error) {
	defer e.Conn.Close()
	trap := make([]byte, 16)
	for {
		_, err := io.ReadFull(e.Conn, trap)
		if err != nil {
			break
		}
	}
	return 0, errors.New("ILLEGAL CONNECTION ABORTED")
}

func Encode(i int) []byte {
	r := make([]byte, 4)
	rand.Read(r)

	enc := make([]byte, 4)
	binary.LittleEndian.PutUint32(enc, uint32(i))

	tmp := make([]byte, 4)
	copy(tmp, r)
	auth := SH256S(append(tmp, HKEY1...))

	tmp = make([]byte, 4)
	copy(tmp, r)
	mask := SH256S(append(tmp, HKEY2...))

	meta := append(r, auth...)
	xorenc := XORBytes(enc, mask)

	return append(meta, xorenc...)
}

func Decode(b []byte) (int, bool) {
	tmp := make([]byte, 4)
	copy(tmp, b[:4])
	auth := SH256S(append(tmp, HKEY1...))

	if !bytes.Equal(auth, b[4:8]) {
		return 0, false
	}

	tmp = make([]byte, 4)
	copy(tmp, b[:4])
	mask := SH256S(append(tmp, HKEY2...))

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

func SH256(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

func SH256S(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[28:]
}
