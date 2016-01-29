package git

import (
	"compress/zlib"
	"crypto/sha1"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Repository struct {
	Path       string
	Bare       bool
	root       string
	pack       *Pack
	packedRefs *PackedRefs
}

func Open(path string) (*Repository, error) {
	path = filepath.Clean(path)
	fi, err := os.Stat(path)
	if err != nil {
		return nil, err
	}
	if !fi.IsDir() {
		return nil, fmt.Errorf("Not a git repository: %s", path)
	}

	repo := &Repository{
		Path: path,
		root: path,
	}
	defer func() {
		if repo != nil {
			repo.openPackedRefs()
		}
	}()
	if strings.HasSuffix(path, ".git") {
		repo.Bare = true
		return repo, nil
	}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file.Name() == ".git" {
			repo.root = filepath.Join(repo.root, ".git")
			return repo, nil
		}
	}
	return nil, fmt.Errorf("Not a git repository: %s", path)
}

func (r *Repository) Object(id SHA1) (Object, error) {
	return r.readObject(id, nil, false)
}

func (r *Repository) Resolve(obj Object) error {
	if obj.Resolved() || obj.SHA1().Empty() {
		return nil
	}
	_, err := r.readObject(obj.SHA1(), obj, false)
	return err
}

func (r *Repository) readObject(id SHA1, obj Object, headerOnly bool) (Object, error) {
	entry, err := r.entry(id)
	if err != nil {
		return nil, err
	}
	defer entry.Close()

	if obj == nil {
		obj = newObject(entry.Type(), id, r)
	}

	if headerOnly {
		return obj, nil
	}
	b, err := entry.ReadAll()
	if err != nil {
		return nil, err
	}
	err = obj.Parse(b)
	return obj, err
}

func (r *Repository) entry(id SHA1) (objectEntry, error) {
	if r.pack == nil {
		if err := r.openPack(); err != nil {
			return nil, err
		}
	}

	if entry, err := r.pack.entry(id); err == nil {
		return entry, err
	}
	return newLooseObjectEntry(r.root, id)
}

func (r *Repository) openPack() error {
	pattern := filepath.Join(r.root, "objects", "pack", "pack-*.pack")
	files, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}
	switch len(files) {
	case 0: // set empty pack
		r.pack = &Pack{idx: &PackIndexV2{}}
	case 1:
		pack, err := OpenPack(files[0])
		if err != nil {
			return err
		}
		r.pack = pack
	default:
		return errors.New("Found more than 1 pack file")
	}
	return nil
}

func (r *Repository) writeObject(typ string, data ObjectData) (id SHA1, err error) {
	var path string
	defer func() {
		if err != nil && path != "" {
			os.Remove(path)
		}
	}()
	if id, path, err = r.writeObjectData(typ, data); err != nil {
		return
	}
	err = r.storeAsObject(path, id)
	return
}

func (r *Repository) writeObjectData(typ string, data ObjectData) (id SHA1, path string, err error) {
	var f *os.File
	if f, err = ioutil.TempFile("", "go-git"); err != nil {
		return
	}
	defer f.Close()
	path = f.Name()

	zw := zlib.NewWriter(f)
	defer zw.Close()
	hash := sha1.New()
	w := io.MultiWriter(hash, zw)

	if _, err = fmt.Fprintf(w, "%s %d%c", typ, data.Size(), 0); err != nil {
		return
	}
	if _, err = io.Copy(w, data); err != nil {
		return
	}
	id = SHA1FromBytes(hash.Sum(nil))
	return
}

func (r *Repository) storeAsObject(path string, id SHA1) error {
	s := id.String()
	dir := filepath.Join(r.root, "objects", s[:2])
	if err := os.MkdirAll(dir, 0777); err != nil {
		return err
	}
	return os.Rename(path, filepath.Join(dir, s[2:]))
}
