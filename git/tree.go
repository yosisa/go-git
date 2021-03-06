package git

import (
	"bytes"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

var scratchbuf = new(bytes.Buffer)

type Tree struct {
	id      SHA1
	repo    *Repository
	Entries []*TreeEntry
	dirty   bool
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

		entry, err := newTreeEntry(mode, name, id, t.repo)
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

func (t *Tree) Find(path string) (*SparseObject, TreeEntryMode, error) {
	parts := splitPath(path)
	tree := t
	var i int
	for ; i < len(parts); i++ {
		if !tree.Resolved() {
			break
		}
		_, entry := tree.findEntry(parts[i])
		if entry == nil {
			return nil, 0, ErrObjectNotFound
		}
		if i == len(parts)-1 {
			return entry.Object, entry.Mode, nil
		}
		if entry.Object.obj == nil {
			tree = t.repo.NewTree()
			tree.id = entry.Object.SHA1()
			continue
		}
		subtree, ok := entry.Object.obj.(*Tree)
		if !ok {
			return nil, 0, ErrObjectNotFound
		}
		tree = subtree
	}
	return tree.fastFind(parts[i:])
}

func (t *Tree) fastFind(parts []string) (*SparseObject, TreeEntryMode, error) {
	id := t.id
	mode := ModeTree
	for _, name := range parts {
		if mode != ModeTree {
			return nil, 0, ErrObjectNotFound
		}
		entry, err := t.repo.entry(id)
		if err != nil {
			return nil, 0, err
		}
		defer entry.Close()

		if entry.Type() != "tree" {
			return nil, 0, ErrObjectNotFound
		}
		data, err := entry.ReadAll()
		if err != nil {
			return nil, 0, err
		}
		id, mode, err = findTreeEntryBytes(data, name)
		if err != nil {
			return nil, 0, err
		}
	}
	return newSparseObject(id, t.repo), mode, nil
}

func findTreeEntryBytes(data []byte, name string) (id SHA1, mode TreeEntryMode, err error) {
	scratchbuf.Reset()
	scratchbuf.WriteString(name)
	scratchbuf.WriteByte(0)
	term := scratchbuf.Bytes()
	for len(data) > 0 {
		var i int
		if data[5] == ' ' {
			i = 5
		} else if data[6] == ' ' {
			i = 6
		} else {
			return id, 0, ErrUnknownFormat
		}
		if mode, err = parseMode(data[:i]); err != nil {
			return
		}
		data = data[i+1:]

		n := len(term)
		if len(data) < n {
			break
		}
		if data[n-1] == 0 && bytes.Equal(data[:n], term) {
			return SHA1FromBytes(data[n : n+20]), mode, nil
		}
		i = bytes.IndexByte(data, 0)
		if i < 0 {
			return id, 0, ErrUnknownFormat
		}
		data = data[i+21:]
	}
	return id, 0, ErrObjectNotFound
}

func (t *Tree) Add(path string, obj Object, mode TreeEntryMode) error {
	dir, name := splitDirBase(path)
	tree, err := t.getSubTree(dir, true)
	if err != nil {
		return err
	}
	tree.addEntry(name, obj, mode)
	return nil
}

func (t *Tree) addEntry(name string, obj Object, mode TreeEntryMode) {
	t.dirty = true
	if _, entry := t.findEntry(name); entry != nil {
		entry.Mode = mode
		entry.Object = &SparseObject{repo: t.repo, obj: obj}
		return
	}
	t.Entries = append(t.Entries, &TreeEntry{
		Mode:   mode,
		Name:   name,
		Object: &SparseObject{repo: t.repo, obj: obj},
	})
}

func (t *Tree) Remove(path string) error {
	dir, name := splitDirBase(path)
	tree, err := t.getSubTree(dir, false)
	if err != nil {
		if err == ErrObjectNotFound {
			return nil
		}
		return err
	}
	tree.removeEntry(name)
	return nil
}

func (t *Tree) removeEntry(name string) {
	if i, entry := t.findEntry(name); entry != nil {
		copy(t.Entries[i:], t.Entries[i+1:])
		t.Entries = t.Entries[:len(t.Entries)-1]
		t.dirty = true
		return
	}
}

func (t *Tree) getSubTree(parts []string, create bool) (*Tree, error) {
	if err := t.Resolve(); err != nil {
		return nil, err
	}
	tree := t
	var i int
	for ; i < len(parts); i++ {
		_, entry := tree.findEntry(parts[i])
		if entry == nil {
			break
		}
		obj, err := entry.Object.Resolve()
		if err != nil {
			return nil, err
		}
		subtree, ok := obj.(*Tree)
		if !ok {
			break
		}
		tree = subtree
	}
	if i == len(parts) {
		return tree, nil
	}
	if !create {
		return nil, ErrObjectNotFound
	}
	return tree.makeSubTrees(parts[i:]), nil
}

func (t *Tree) makeSubTrees(parts []string) *Tree {
	tree := t
	for _, name := range parts {
		newTree := t.repo.NewTree()
		tree.addEntry(name, newTree, ModeTree)
		tree = newTree
	}
	return tree
}

func (t *Tree) findEntry(name string) (int, *TreeEntry) {
	for i, entry := range t.Entries {
		if entry.Name == name {
			return i, entry
		}
	}
	return 0, nil
}

func (t *Tree) Write() error {
	_, err := t.write()
	return err
}

// write walks subtrees to check and save dirty objects recursively. To save
// entire tree correctly, it's necessary to save objects from leaf to root. If
// something changed in subtrees, the parent tree also need to be saved.
func (t *Tree) write() (bool, error) {
	for _, entry := range t.Entries {
		// It's safe to ignore unresolved objects because it's stored in
		// the repository and not modified.
		if entry.Object.obj == nil {
			continue
		}
		if subtree, ok := entry.Object.obj.(*Tree); ok {
			if changed, err := subtree.write(); err != nil {
				return false, err
			} else if changed {
				t.dirty = true
			}
		} else if entry.Object.SHA1().Empty() {
			if err := entry.Object.Write(); err != nil {
				return false, err
			}
			t.dirty = true
		}
	}
	if !t.dirty {
		return false, nil
	}

	sort.Sort(ByName(t.Entries))
	b := new(bytes.Buffer)
	for _, entry := range t.Entries {
		if entry.Object.obj != nil {
			if subtree, ok := entry.Object.obj.(*Tree); ok && len(subtree.Entries) == 0 {
				continue // No need to write empty tree object
			}
		}
		fmt.Fprintf(b, "%s %s%c", entry.Mode, entry.Name, 0)
		b.Write(entry.Object.SHA1().Bytes())
	}
	id, err := t.repo.writeObject("tree", bytes.NewReader(b.Bytes()))
	if err != nil {
		return false, err
	}
	t.id = id
	t.dirty = false
	return true, nil
}

type TreeEntry struct {
	Mode   TreeEntryMode
	Name   string
	Object *SparseObject
}

func newTreeEntry(mode, name, id []byte, repo *Repository) (*TreeEntry, error) {
	m, err := parseMode(mode)
	if err != nil {
		return nil, err
	}
	entry := &TreeEntry{
		Mode:   m,
		Name:   string(name),
		Object: newSparseObject(SHA1FromBytes(id), repo),
	}
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
	ModeFile    TreeEntryMode = 0100644
	ModeFileEx  TreeEntryMode = 0100755
	ModeSymlink TreeEntryMode = 0120000
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

func splitPath(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

func splitDirBase(path string) ([]string, string) {
	s := splitPath(path)
	return s[:len(s)-1], s[len(s)-1]
}

func (r *Repository) NewTree() *Tree {
	return &Tree{repo: r}
}

type ByName []*TreeEntry

func (z ByName) Len() int           { return len(z) }
func (z ByName) Swap(i, j int)      { z[i], z[j] = z[j], z[i] }
func (z ByName) Less(i, j int) bool { return z[i].canonicalName() < z[j].canonicalName() }
