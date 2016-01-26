package git

import "testing"

func TestTreeEntryMode(t *testing.T) {
	m, err := parseMode([]byte("100644"))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if m != 0100644 {
		t.Fatalf("%d != 0100644", m)
	}
	if s := m.String(); s != "100644" {
		t.Fatalf("%s != 100644", s)
	}
}
