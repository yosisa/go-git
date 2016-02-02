package git

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

type Ref struct {
	repo   *Repository
	Name   string
	SHA1   SHA1
	commit *SHA1
}

func (r *Ref) Write() error {
	path := filepath.Join(r.repo.root, r.Name)
	if err := os.MkdirAll(filepath.Dir(path), 0777); err != nil {
		return err
	}
	return ioutil.WriteFile(path, []byte(r.SHA1.String()+"\n"), 0666)
}

// Commit returns a commit object that the ref points to. It also understand
// annotated tags. The returned commit object maybe unresolved, it's necessary
// to call Resolve function before using commit data.
func (r *Ref) Commit() (*Commit, error) {
	id := r.SHA1
	if r.commit != nil {
		id = *r.commit
	}
	obj, err := r.repo.Object(id)
	if err != nil {
		return nil, err
	}
	for {
		switch typed := obj.(type) {
		case *Commit:
			return typed, nil
		case *Tag:
			if c, ok := typed.Object.(*Commit); ok {
				return c, nil
			}
			obj = typed.Object
			if err = obj.Resolve(); err != nil {
				return nil, err
			}
		default:
			return nil, ErrTypeMismatch
		}
	}
}

func (r *Repository) NewRef(name string, id SHA1) *Ref {
	return &Ref{
		repo: r,
		Name: name,
		SHA1: id,
	}
}

type Refs []*Ref

func (refs Refs) merge(other []*Ref) []*Ref {
	m := refsToMap(refs)
	for _, ref := range other {
		m[ref.Name] = ref
	}
	return mapToRefs(m)
}

func (refs Refs) find(suffix string) *Ref {
	for _, ref := range refs {
		if strings.HasSuffix(ref.Name, suffix) {
			return ref
		}
	}
	return nil
}

func (r *Repository) Ref(name string) (*Ref, error) {
	if ref, err := r.looseRef(name); err == nil {
		return ref, nil
	}
	if ref := r.packedRefs.Ref(name); ref != nil {
		return ref, nil
	}
	return nil, fmt.Errorf("Ref not found: %s", name)
}

func (r *Repository) looseRef(name string) (*Ref, error) {
	f, err := os.Open(filepath.Join(r.root, name))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	b, err := buf.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	return r.NewRef(name, SHA1FromHex(b[:len(b)-1])), nil
}

func (r *Repository) Branches() []*Ref {
	prefix := filepath.Join("refs", "heads")
	refs := r.packedRefs.Refs(prefix)
	loose, err := r.looseRefs(prefix)
	if err != nil {
		return nil
	}
	return Refs(refs).merge(loose)
}

func (r Repository) Tags() []*Ref {
	prefix := filepath.Join("refs", "tags")
	refs := r.packedRefs.Refs(prefix)
	loose, err := r.looseRefs(prefix)
	if err != nil {
		return nil
	}
	return Refs(refs).merge(loose)
}

func (r Repository) looseRefs(path string) ([]*Ref, error) {
	path = filepath.Join(r.root, path)
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}
	var refs []*Ref
	for _, file := range files {
		name := filepath.Join(path, file.Name())
		if ref, err := r.Ref(name); err == nil {
			refs = append(refs, ref)
		}
	}
	return refs, nil
}

func (r *Repository) Head() (*Ref, error) {
	f, err := os.Open(filepath.Join(r.root, "HEAD"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	buf := bufio.NewReader(f)
	b, err := buf.ReadBytes('\n')
	if err != nil {
		return nil, err
	}
	if !bytes.HasPrefix(b, []byte("ref: ")) {
		return nil, ErrUnknownFormat
	}
	return r.Ref(string(b[5 : len(b)-1]))
}

type PackedRefs struct {
	repo *Repository
	Path string
	Err  error
	refs map[string]*Ref
}

func (p *PackedRefs) Ref(name string) *Ref {
	if p.refs == nil {
		if p.Err = p.Parse(); p.Err != nil {
			return nil
		}
	}
	return p.refs[name]
}

func (p *PackedRefs) Refs(prefix string) []*Ref {
	if p.refs == nil {
		if p.Err = p.Parse(); p.Err != nil {
			return nil
		}
	}
	var out []*Ref
	for _, ref := range p.refs {
		if strings.HasPrefix(ref.Name, prefix) {
			out = append(out, ref)
		}
	}
	return out
}

func (p *PackedRefs) Parse() error {
	p.refs = make(map[string]*Ref)
	f, err := os.Open(p.Path)
	if err != nil {
		return err
	}
	defer f.Close()

	var ref *Ref
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := scan.Bytes()
		if pos := bytes.IndexByte(line, '#'); pos != -1 {
			line = line[:pos]
		}
		bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		if line[0] == '^' {
			if ref == nil {
				return ErrUnknownFormat
			}
			commit := SHA1FromHex(line[1:])
			ref.commit = &commit
			continue
		}
		items := bytes.Split(line, []byte{' '})
		if len(items) != 2 {
			return ErrUnknownFormat
		}
		name := string(items[1])
		ref = p.repo.NewRef(name, SHA1FromHex(items[0]))
		p.refs[name] = ref
	}
	if err := scan.Err(); err != nil {
		return err
	}
	return nil
}

func (r *Repository) openPackedRefs() {
	r.packedRefs = &PackedRefs{
		repo: r,
		Path: filepath.Join(r.root, "packed-refs"),
	}
}

func refsToMap(refs []*Ref) map[string]*Ref {
	out := make(map[string]*Ref)
	for _, ref := range refs {
		out[ref.Name] = ref
	}
	return out
}

func mapToRefs(m map[string]*Ref) (refs []*Ref) {
	for _, ref := range m {
		refs = append(refs, ref)
	}
	return
}
