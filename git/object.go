package git

import (
	"errors"
	"io"
)

type Object interface {
	SHA1() SHA1
	Parse([]byte) error
	Resolve() error
	Resolved() bool
}

func newObject(typ string, id SHA1, repo *Repository) Object {
	switch typ {
	case "blob":
		return newBlob(id, repo)
	case "tree":
		return newTree(id, repo)
	case "commit":
		return newCommit(id, repo)
	case "tag":
		return newTag(id, repo)
	}
	panic("Unknown object type: " + typ)
}

func objectType(obj Object) (string, error) {
	switch obj.(type) {
	case *Blob:
		return "blob", nil
	case *Tree:
		return "tree", nil
	case *Commit:
		return "commit", nil
	case *Tag:
		return "tag", nil
	}
	return "", errors.New("Unknown object")
}

type objectEntry interface {
	Type() string
	ReadAll() ([]byte, error)
	Close() error
}

type SparseObject struct {
	SHA1 SHA1
	obj  Object
	err  error
	repo *Repository
}

func newSparseObject(id SHA1, repo *Repository) *SparseObject {
	return &SparseObject{
		SHA1: id,
		repo: repo,
	}
}

func (s *SparseObject) Resolve() (Object, error) {
	if s.obj == nil && s.err == nil {
		s.obj, s.err = s.repo.Object(s.SHA1)
	}
	return s.obj, s.err
}

type ObjectData interface {
	io.Reader
	Size() int64
}
