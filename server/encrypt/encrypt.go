package encrypt

import (
	"bytes"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"golang.org/x/crypto/nacl/secretbox"
	"io"
	"log"
	mr "math/rand"
	"net"
	"time"
)

const (
	TsRng int = 180
)

type Keyring struct {
	p    *[32]byte
	keys [][]byte
}

func NewKeyring(s string) *Keyring {
	b := SH256L([]byte(s))
	pw := sha256.Sum256(b)
	k := make([][]byte, 6)
	for i, _ := range k {
		k[i] = SH256L(b[(3 * i):(12 + 3*i)])
	}
	return &Keyring{
		keys: k,
		p:    &pw,
	}
}

type EncStreamServer struct {
	net.Conn
	keyring        *Keyring
	rBuf, dBuf     []byte
	rNonce, sNonce [24]byte
}

func NewEncStreamServer(conn net.Conn, k *Keyring) *EncStreamServer {
	e := &EncStreamServer{
		Conn:    conn,
		keyring: k,
		dBuf:    make([]byte, 16),
	}
	copy(e.rNonce[:8], k.keys[3][24:])
	copy(e.sNonce[:8], k.keys[3][:8])
	return e
}

func (e *EncStreamServer) Read(b []byte) (int, error) {
	if len(e.rBuf) > 0 {
		n := copy(b, e.rBuf)
		e.rBuf = e.rBuf[n:]
		return n, nil
	}

	if n, err := io.ReadFull(e.Conn, e.dBuf); err != nil || n != 16 {
		return 0, err
	}

	size, ok := ServerDecode(e.dBuf, e.keyring.keys)
	if !ok {
		log.Println("INVALID PACKET RECEIVED")
		return e.Drop()
	}

	c := make([]byte, size)
	if _, err := io.ReadFull(e.Conn, c); err != nil {
		log.Printf("%v", err)
		return 0, err
	}

	p, ok := secretbox.Open(nil, c[:size], &e.rNonce, e.keyring.p)
	if !ok {
		log.Println("DECRYPTION FAILED")
		return e.Drop()
	}
	increment(&e.rNonce)

	n := copy(b, p)
	if n < len(p) {
		e.rBuf = p[n:]
	}

	return n, nil
}

func (e *EncStreamServer) Write(b []byte) (int, error) {
	sidx, eidx, chnk := 0, 0, Chunk()
	for ; sidx < len(b); sidx = eidx {
		if len(b)-eidx >= chnk {
			eidx += chnk
		} else {
			eidx = len(b)
		}
		cipher := secretbox.Seal([]byte{}, b[sidx:eidx], &e.sNonce, e.keyring.p)
		increment(&e.sNonce)
		enc := ServerEncode(len(cipher), e.keyring.keys)
		if _, err := e.Conn.Write(enc); err != nil {
			return sidx, err
		}
		if _, err := e.Conn.Write(cipher); err != nil {
			return sidx, err
		}
	}
	return sidx, nil
}

func (e *EncStreamServer) Close() error {
	return e.Conn.Close()
}

func (e *EncStreamServer) Drop() (int, error) {
	defer e.Conn.Close()
	trap := make([]byte, 16)
	for {
		_, err := io.ReadFull(e.Conn, trap)
		if err != nil {
			break
		}
	}
	return 0, errors.New("ILLEGAL CONNECTION CLOSED")
}

func ServerEncode(i int, keys [][]byte) []byte {
	head := make([]byte, 8)
	iBuf := make([]byte, 4)
	hBuf := make([]byte, 36)

	rand.Read(head[:2])
	copy(hBuf[:2], head[:2])

	binary.LittleEndian.PutUint32(iBuf, uint32(i))
	copy(hBuf[4:], keys[4])
	copy(head[2:6], XORBytes(iBuf, SH256S(hBuf)))

	copy(hBuf[:6], head[:6])
	copy(hBuf[6:], keys[5])
	copy(head[6:8], SH256SS(hBuf))

	return head
}

func ServerDecode(b []byte, keys [][]byte) (int, bool) {
	hBuf := make([]byte, 36)
	copy(hBuf[:4], b[:4])

	copy(hBuf[4:], keys[2])
	iBuf := XORBytes(b[4:8], SH256S(hBuf))
	if Abs(int(time.Now().Unix())-int(binary.LittleEndian.Uint32(iBuf))) > TsRng {
		log.Println("INCORRECT TIMESTAMP")
		return 0, false
	}

	copy(hBuf[4:], keys[1])
	iBuf = XORBytes(b[8:12], SH256S(hBuf))
	i := int(binary.LittleEndian.Uint32(iBuf))

	copy(hBuf[:12], b[:12])
	copy(hBuf[12:], keys[0][:24])
	if !bytes.Equal(b[12:16], SH256S(hBuf)) {
		return 0, false
	}

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
	return s[16:20]
}

func SH256SS(b []byte) []byte {
	s := SH256L(b)
	return s[22:24]
}

func Abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func Chunk() int {
	mr.Seed(time.Now().UnixNano())
	c, x := 16384, mr.Intn(512)
	if mr.Intn(2) == 0 {
		c += x
	} else {
		c -= x
	}
	return c
}
