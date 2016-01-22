package git

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/yosisa/go-git/lru"
)

type Tree struct {
	id      SHA1
	repo    *Repository
	Entries []*TreeEntry
}

func newTree(id SHA1, repo *Repository) *Tree {
	return &Tree{
		id:   id,
		repo: repo,
	}
}

func (t *Tree) SHA1() SHA1 {
	return t.id
}

func (t *Tree) Parse(data []byte) error {
	var mode, name, id, rest []byte
	var pos int
	for len(data) > 0 {
		if pos = bytes.IndexByte(data, ' '); pos == -1 {
			return ErrUnknownFormat
		}
		mode, rest = data[:pos], data[pos+1:]

		if pos = bytes.IndexByte(rest, 0); pos == -1 {
			return ErrUnknownFormat
		}
		name, id, rest = rest[:pos], rest[pos+1:pos+21], rest[pos+21:]

		last := len(mode) + len(name) + 22
		entry, err := newTreeEntry(mode, name, id, data[:last], t.repo)
		if err != nil {
			return err
		}
		t.Entries = append(t.Entries, entry)
		data = rest
	}
	return nil
}

func (t *Tree) Resolve() error {
	return t.repo.Resolve(t)
}

func (t *Tree) Resolved() bool {
	return t.Entries != nil
}

func (t *Tree) Find(path string) (*SparseObject, error) {
	path = strings.TrimLeft(path, "/")
	return t.find(strings.Split(path, "/"))
}

func (t *Tree) find(items []string) (*SparseObject, error) {
	if err := t.repo.Resolve(t); err != nil {
		return nil, err
	}
	for _, e := range t.Entries {
		if e.Name == items[0] {
			if len(items) == 1 {
				return e.Object, nil
			}
			obj, err := e.Object.Resolve()
			if err != nil {
				return nil, err
			}
			if tree, ok := obj.(*Tree); ok {
				return tree.find(items[1:])
			}
			break
		}
	}
	return nil, ErrObjectNotFound
}

func (t *Tree) AddEntry(name string, id SHA1, mode TreeEntryMode) {
	for _, entry := range t.Entries {
		if entry.Name == name {
			entry.Mode = mode
			entry.Object.SHA1 = id
			return
		}
	}
	t.Entries = append(t.Entries, &TreeEntry{
		Mode:   mode,
		Name:   name,
		Object: newSparseObject(id, t.repo),
	})
}

func (t *Tree) Write() error {
	sort.Sort(ByName(t.Entries))
	b := new(bytes.Buffer)
	for _, entry := range t.Entries {
		fmt.Fprintf(b, "%s %s%c", entry.Mode, entry.Name, 0)
		b.Write(entry.Object.SHA1[:])
	}
	id, err := t.repo.writeObject("tree", bytes.NewReader(b.Bytes()))
	if err == nil {
		t.id = id
	}
	return err
}

var treeEntryCache = lru.New(1 << 16)

type TreeEntry struct {
	Mode   TreeEntryMode
	Name   string
	Object *SparseObject
}

func newTreeEntry(mode, name, id, row []byte, repo *Repository) (*TreeEntry, error) {
	key := string(row)
	if entry, ok := treeEntryCache.Get(key); ok {
		return entry.(*TreeEntry), nil
	}
	m, err := parseMode(mode)
	if err != nil {
		return nil, err
	}
	entry := &TreeEntry{
		Mode:   m,
		Name:   string(name),
		Object: newSparseObject(SHA1FromBytes(id), repo),
	}
	treeEntryCache.Add(key, entry)
	return entry, nil
}

func (t *TreeEntry) Size() int {
	return 8 + len(t.Name)
}

func (t *TreeEntry) canonicalName() string {
	if t.Mode&ModeTree != 0 {
		return t.Name + "/"
	}
	return t.Name
}

type TreeEntryMode uint32

const (
	ModeTree    TreeEntryMode = 0040000
	ModeFile                  = 0100644
	ModeFileEx                = 0100755
	ModeSymlink               = 0120000
)

func parseMode(bs []byte) (TreeEntryMode, error) {
	var mode TreeEntryMode
	for _, b := range bs {
		n := b - 0x30
		if n < 0 || n > 7 {
			return 0, fmt.Errorf("%d not in octal range", n)
		}
		mode = mode<<3 | TreeEntryMode(n)
	}
	return mode, nil
}

func (m TreeEntryMode) String() string {
	var s string
	for m > 0 {
		n := int(m & 0x7)
		s = strconv.Itoa(n) + s
		m = m >> 3
	}
	return s
}

func (r *Repository) NewTree() *Tree {
	return &Tree{repo: r}
}

type ByName []*TreeEntry

func (z ByName) Len() int           { return len(z) }
func (z ByName) Swap(i, j int)      { z[i], z[j] = z[j], z[i] }
func (z ByName) Less(i, j int) bool { return z[i].canonicalName() < z[j].canonicalName() }
