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
	"time"
)

const (
	Chunk    int = 16384
	ValidRng int = 180
)

type Key struct {
	hash     *[32]byte
	salt     []byte
	pepper   []byte
	cinnamon []byte
}

func NewKey(s string) *Key {
	h := sha256.Sum256([]byte(s))
	return &Key{
		hash:     &h,
		salt:     SH256L(h[:12]),
		pepper:   SH256L(h[4:16]),
		cinnamon: SH256L(h[8:20]),
	}
}

type EncStream struct {
	net.Conn
	key            *Key
	rBuf, dBuf     []byte
	rNonce, sNonce [24]byte
}

func NewEncStream(conn net.Conn, k *Key) *EncStream {
	e := &EncStream{
		Conn: conn,
		key:  k,
		dBuf: make([]byte, 16),
	}
	copy(e.rNonce[:4], SH256S(k.hash[:]))
	copy(e.sNonce[:4], SH256S(k.hash[:]))
	return e
}

func (e *EncStream) Read(b []byte) (int, error) {
	if len(e.rBuf) > 0 {
		n := copy(b, e.rBuf)
		e.rBuf = e.rBuf[n:]
		return n, nil
	}

	if n, err := io.ReadFull(e.Conn, e.dBuf); err != nil || n != 16 {
		return 0, err
	}

	size, ok := Decode(e.dBuf, e.key.salt, e.key.pepper, e.key.cinnamon)
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
		enc := Encode(len(cipher), e.key.salt, e.key.pepper, e.key.cinnamon)
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
	trap := make([]byte, 16)
	for {
		_, err := io.ReadFull(e.Conn, trap)
		if err != nil {
			break
		}
	}
	return 0, errors.New("ILLEGAL CONNECTION")
}

func Encode(i int, salt, pepper, cinnamon []byte) []byte {
	iBuf := make([]byte, 4)
	head := make([]byte, 16)
	hBuf := make([]byte, 36)

	rand.Read(head[:4])

	copy(hBuf[:4], head[:4])
	copy(hBuf[4:], salt)
	copy(head[4:8], SH256S(hBuf))

	t := time.Now().Unix()
	binary.LittleEndian.PutUint32(iBuf, uint32(t))
	copy(hBuf[4:], cinnamon)
	copy(head[8:12], XORBytes(iBuf, SH256S(hBuf)))

	binary.LittleEndian.PutUint32(iBuf, uint32(i))
	copy(hBuf[4:], pepper)
	copy(head[12:], XORBytes(iBuf, SH256S(hBuf)))

	return head
}

func Decode(b, salt, pepper, cinnamon []byte) (int, bool) {
	hBuf := make([]byte, 36)

	copy(hBuf[:4], b[:4])
	copy(hBuf[4:], salt)
	if !bytes.Equal(b[4:8], SH256S(hBuf)) {
		return 0, false
	}

	copy(hBuf[4:], cinnamon)
	iBuf := XORBytes(b[8:12], SH256S(hBuf))
	if Abs(int(time.Now().Unix())-int(binary.LittleEndian.Uint32(iBuf))) > ValidRng {
		log.Println("EXPIRED TIMESTAMP")
		return 0, false
	}

	copy(hBuf[4:], pepper)
	iBuf = XORBytes(b[12:16], SH256S(hBuf))
	i := int(binary.LittleEndian.Uint32(iBuf))

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
	for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
		s[i], s[j] = s[j], s[i]
	}
	return s[:]
}

func SH256S(b []byte) []byte {
	s := SH256L(b)
	s[0], s[1], s[2], s[3] = s[6], s[12], s[18], s[24]
	return s[:4]
}

func Abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}
