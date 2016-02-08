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
	Write() error
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
	id   SHA1
	obj  Object
	err  error
	repo *Repository
}

func newSparseObject(id SHA1, repo *Repository) *SparseObject {
	return &SparseObject{
		id:   id,
		repo: repo,
	}
}

func (s *SparseObject) SHA1() SHA1 {
	if s.obj != nil {
		return s.obj.SHA1()
	}
	return s.id
}

func (s *SparseObject) Resolve() (Object, error) {
	if s.obj == nil && s.err == nil {
		s.obj, s.err = s.repo.Object(s.id)
	}
	return s.obj, s.err
}

func (s *SparseObject) Write() error {
	if s.obj == nil {
		return nil
	}
	if err := s.obj.Write(); err != nil {
		return err
	}
	s.id = s.obj.SHA1()
	return nil
}

type ObjectData interface {
	io.Reader
	Size() int64
}
