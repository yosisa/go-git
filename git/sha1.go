package git

import (
	"bytes"
	"encoding/hex"
	"io"
)

type SHA1 [20]byte

var emptySHA1 SHA1

func (b SHA1) Bytes() []byte {
	return b[:]
}

func (b SHA1) String() string {
	return hex.EncodeToString(b[:])
}

func (b SHA1) Compare(other SHA1) int {
	return bytes.Compare(b[:], other[:])
}

func (b *SHA1) Fill(r io.Reader) error {
	_, err := io.ReadFull(r, (*b)[:])
	return err
}

func (b SHA1) Empty() bool {
	return b == emptySHA1
}

func NewSHA1(s string) (sha SHA1, err error) {
	var b []byte
	if b, err = hex.DecodeString(s); err == nil {
		copy(sha[:], b)
	}
	return
}

func SHA1FromHex(b []byte) (sha SHA1) {
	if _, err := hex.Decode(sha[:], b); err != nil {
		panic(err)
	}
	return
}

func SHA1FromHexString(s string) SHA1 {
	sha, err := NewSHA1(s)
	if err != nil {
		panic(err)
	}
	return sha
}

func SHA1FromBytes(b []byte) (sha SHA1) {
	copy(sha[:], b)
	return
}

func readSHA1(r io.Reader) (sha SHA1, err error) {
	_, err = io.ReadFull(r, sha[:])
	return
}
