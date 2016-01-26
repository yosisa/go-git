package git

import "errors"

var ErrTypeMismatch = errors.New("Error type mismatch")

func (r *Repository) Blob(id SHA1) (*Blob, error) {
	return asBlob(r.Object(id))
}

func (r *Repository) Tree(id SHA1) (*Tree, error) {
	return asTree(r.Object(id))
}

func (r *Repository) Commit(id SHA1) (*Commit, error) {
	return asCommit(r.Object(id))
}

func (r *Repository) Tag(id SHA1) (*Tag, error) {
	return asTag(r.Object(id))
}

func (s *SparseObject) Blob() (*Blob, error) {
	return asBlob(s.Resolve())
}

func (s *SparseObject) Tree() (*Tree, error) {
	return asTree(s.Resolve())
}

func (s *SparseObject) Commit() (*Commit, error) {
	return asCommit(s.Resolve())
}

func (s *SparseObject) Tag() (*Tag, error) {
	return asTag(s.Resolve())
}

func (t *Tree) FindBlob(path string) (*Blob, error) {
	sobj, err := t.Find(path)
	if err != nil {
		return nil, err
	}
	return sobj.Blob()
}

func (t *Tree) FindTree(path string) (*Tree, error) {
	sobj, err := t.Find(path)
	if err != nil {
		return nil, err
	}
	return sobj.Tree()
}

func asBlob(obj Object, err error) (*Blob, error) {
	if err != nil {
		return nil, err
	}
	if blob, ok := obj.(*Blob); ok {
		return blob, nil
	}
	return nil, ErrTypeMismatch
}

func asTree(obj Object, err error) (*Tree, error) {
	if err != nil {
		return nil, err
	}
	if tree, ok := obj.(*Tree); ok {
		return tree, nil
	}
	return nil, ErrTypeMismatch
}

func asCommit(obj Object, err error) (*Commit, error) {
	if err != nil {
		return nil, err
	}
	if commit, ok := obj.(*Commit); ok {
		return commit, nil
	}
	return nil, ErrTypeMismatch
}

func asTag(obj Object, err error) (*Tag, error) {
	if err != nil {
		return nil, err
	}
	if tag, ok := obj.(*Tag); ok {
		return tag, nil
	}
	return nil, ErrTypeMismatch
}
