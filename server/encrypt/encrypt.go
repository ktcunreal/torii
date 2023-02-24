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
	TsRng int = 300
)

type Keyring struct {
	k_cipher *[32]byte
	k_nonce []byte
	k_client []byte
	k_server []byte
	k_timestamp []byte
	k_packet_len []byte
	k_chksum []byte
}

func NewKeyring(s string) *Keyring {
	k_nonce := SH256L([]byte("k_nonce_" + s))
	k_client := SH256L([]byte("k_client_" + s))
	k_server := SH256L([]byte("k_server_" + s))
	k_timestamp := SH256L([]byte("k_timestamp_" + s))
	k_chksum := SH256L([]byte("k_chksum_" + s))
	k_cipher := sha256.Sum256([]byte("k_cipher_" + s))
	return &Keyring{
		k_nonce: k_nonce,
		k_client: k_client,
		k_server: k_server,
		k_timestamp: k_timestamp,
		k_chksum: 	k_chksum,
		k_cipher: &k_cipher,
	}
}

type EncStreamServer struct {
	net.Conn
	keyring        *Keyring
	rBuf, dBuf     []byte
	rNonce, sNonce [24]byte
}

func NewEncStreamServer(conn net.Conn, keys *Keyring) *EncStreamServer {
	e := &EncStreamServer{
		Conn:    conn,
		keyring: keys,
		dBuf:    make([]byte, 16),
	}
	copy(e.rNonce[:8], keys.k_nonce[24:])
	copy(e.sNonce[:8], keys.k_nonce[:8])
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

	size, ok := ServerDecode(e.dBuf, e.keyring)
	if !ok {
		log.Println("INVALID PACKET RECEIVED")
		return e.Drop()
	}

	c := make([]byte, size)
	if _, err := io.ReadFull(e.Conn, c); err != nil {
		log.Printf("%v", err)
		return 0, err
	}

	p, ok := secretbox.Open(nil, c[:size], &e.rNonce, e.keyring.k_cipher)
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
		cipher := secretbox.Seal([]byte{}, b[sidx:eidx], &e.sNonce, e.keyring.k_cipher)
		increment(&e.sNonce)
		enc := ServerEncode(len(cipher), e.keyring)
		if _, err := e.Conn.Write(append(enc, cipher...)); err != nil {
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

func ServerEncode(i int, keys *Keyring) []byte {
	head := make([]byte, 8)
	iBuf := make([]byte, 4)
	hBuf := make([]byte, 36)

	rand.Read(head[:4])
	copy(hBuf[:4], head[:4])

	binary.LittleEndian.PutUint32(iBuf, uint32(i))
	copy(hBuf[4:], keys.k_server)
	copy(head[4:8], XORBytes(iBuf, SH256S(hBuf)))

	return head
}

func ServerDecode(b []byte, keys *Keyring) (int, bool) {
	hBuf := make([]byte, 36)
	copy(hBuf[:4], b[:4])

	copy(hBuf[4:], keys.k_timestamp)
	iBuf := XORBytes(b[4:8], SH256S(hBuf))
	if Abs(int(time.Now().Unix())-int(binary.LittleEndian.Uint32(iBuf))) > TsRng {
		log.Println("INCORRECT TIMESTAMP")
		return 0, false
	}

	copy(hBuf[4:8], b[4:8])
	copy(hBuf[8:], keys.k_client[4:])
	iBuf = XORBytes(b[8:12], SH256S(hBuf))
	i := int(binary.LittleEndian.Uint32(iBuf))

	copy(hBuf[:12], b[:12])
	copy(hBuf[12:], keys.k_chksum[:24])
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
	return s[:]
}

func SH256S(b []byte) []byte {
	s := SH256L(b)
	return s[16:20]
}

func Abs(i int) int {
	if i < 0 {
		return -i
	}
	return i
}

func Chunk() int {
	mr.Seed(time.Now().UnixNano())
	return 20480 - mr.Intn(10240)
}
