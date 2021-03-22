package utils

import (
	"crypto/sha256"
	"time"
)

type MaskParam struct {
	key          []byte
	current_ts   []byte
	lapsed_ts    []byte
	current_auth []byte
	lapsed_auth  []byte
	current_xor  []byte
	lapsed_xor   []byte
}

func (m *MaskParam) SetKey(k []byte) {
	m.key = SH256L(k)
	m.Update()
}

func (m *MaskParam) Update() {
	m.lapsed_ts = m.current_ts
	m.current_ts = []byte(time.Now().Format(f))

	m.lapsed_auth = m.current_auth
	m.current_auth = SH256S(append(m.current_ts, m.key...))

	m.lapsed_xor = m.current_xor
	m.current_xor = SH256S(append(m.key, m.current_ts...))
}

func SH256L(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[:]
}

func SH256S(b []byte) []byte {
	s := sha256.Sum256(b)
	return s[28:]
}
