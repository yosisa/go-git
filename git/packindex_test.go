package git

import "testing"

func TestEntry(t *testing.T) {
	idx := new(PackIndexV2)
	idx.Fanout[0] = 1
	idx.Fanout[1] = 3
	idx.Objects = []SHA1{
		SHA1FromHexString("0010000000000000000000000000000000000000"),
		SHA1FromHexString("0100000000000000000000000000000000000000"),
		SHA1FromHexString("0110000000000000000000000000000000000000"),
	}
	idx.Offsets = []uint32{
		0,
		1,
		2,
	}

	id := SHA1FromHexString("0100000000000000000000000000000000000000")
	entry := idx.Entry(id)
	if entry == nil {
		t.Errorf("not found")
	} else if entry.Offset != 1 {
		t.Errorf("unexpected offset: %d", entry.Offset)
	}

	id = SHA1FromHexString("0100000000000000000000000000000000000001")
	entry = idx.Entry(id)
	if entry != nil {
		t.Errorf("unexpected match: %s %d", entry.ID, entry.Offset)
	}
}
