package git

import (
	"bufio"
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"io"
	"os"
	"sort"
)

var packIndexV2Magic = [4]byte{0xff, 't', 'O', 'c'}

type CRC32 [4]byte

type PackIndexV2Header struct {
	Magic   [4]byte
	Version uint32
	Fanout  [256]uint32
}

type PackIndexV2 struct {
	PackIndexV2Header
	Objects       []SHA1
	CRC32s        []CRC32
	Offsets       []uint32
	LargeOffsets  []uint64
	PackFileHash  SHA1
	PackIndexHash SHA1
}

func (idx *PackIndexV2) Parse(r io.Reader) (err error) {
	hasher := sha1.New()
	r = io.TeeReader(r, hasher)

	if err = binary.Read(r, binary.BigEndian, &idx.PackIndexV2Header); err != nil {
		return
	}
	if idx.Magic != packIndexV2Magic || idx.Version != 2 {
		return ErrUnknownFormat
	}

	total := int(idx.Fanout[255])
	idx.Objects = make([]SHA1, total, total)
	for i := 0; i < total; i++ {
		if err = idx.Objects[i].Fill(r); err != nil {
			return
		}
	}

	idx.CRC32s = make([]CRC32, total, total)
	for i := 0; i < total; i++ {
		if _, err = io.ReadFull(r, idx.CRC32s[i][:]); err != nil {
			return
		}
	}

	idx.Offsets = make([]uint32, total, total)
	if err = binary.Read(r, binary.BigEndian, idx.Offsets); err != nil {
		return
	}

	var largeOffsets int
	for _, offset := range idx.Offsets {
		if (offset >> 31) == 1 {
			largeOffsets++
		}
	}
	idx.LargeOffsets = make([]uint64, largeOffsets, largeOffsets)
	if err = binary.Read(r, binary.BigEndian, idx.LargeOffsets); err != nil {
		return
	}

	if err = idx.PackFileHash.Fill(r); err != nil {
		return
	}

	checksum := hasher.Sum(nil)
	if err = idx.PackIndexHash.Fill(r); err != nil {
		return
	}
	if !bytes.Equal(checksum, idx.PackIndexHash[:]) {
		return ErrChecksum
	}
	return
}

func (idx *PackIndexV2) Entry(id SHA1) *PackIndexEntry {
	lower := 0
	if id[0] != 0 {
		lower = int(idx.Fanout[int(id[0])-1])
	}
	upper := int(idx.Fanout[int(id[0])])
	entries := idx.Objects[lower:upper]

	var found bool
	x := sort.Search(len(entries), func(i int) bool {
		n := entries[i].Compare(id)
		if n == 0 {
			found = true
		}
		return n >= 0
	})
	if !found {
		return nil
	}
	x += lower
	return &PackIndexEntry{
		ID:     id,
		Offset: int64(idx.Offsets[x]),
	}
}

type PackIndexEntry struct {
	ID     SHA1
	Offset int64
}

func OpenPackIndex(path string) (*PackIndexV2, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	magic, err := buf.Peek(4)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(magic, packIndexV2Magic[:]) {
		return nil, ErrUnknownFormat
	}
	idx := new(PackIndexV2)
	err = idx.Parse(buf)
	return idx, err
}
