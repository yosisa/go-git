package git

import (
	"bytes"
	"fmt"
)

type Tag struct {
	id     SHA1
	repo   *Repository
	Object Object
	Name   string
	Tagger *User
	Data   []byte
}

func newTag(id SHA1, repo *Repository) *Tag {
	return &Tag{
		id:   id,
		repo: repo,
	}
}

func (t *Tag) SHA1() SHA1 {
	return t.id
}

func (t *Tag) Parse(data []byte) error {
	var err error
	kv := make(map[string][]byte)
	if kv["object"], data, err = readKV(data, "object "); err != nil {
		return err
	}
	if kv["type"], data, err = readKV(data, "type "); err != nil {
		return err
	}
	if kv["tag"], data, err = readKV(data, "tag "); err != nil {
		return err
	}
	if kv["tagger"], data, err = readKV(data, "tagger "); err != nil {
		return err
	}

	obj := newObject(string(kv["type"]), SHA1FromHex(kv["object"]), t.repo)
	tagger, err := newUser(kv["tagger"])
	if err != nil {
		return err
	}
	t.Object = obj
	t.Tagger = tagger
	t.Name = string(kv["tag"])
	t.Data = cloneBytes(data[1:])
	return nil
}

func (t *Tag) Resolve() error {
	return t.repo.Resolve(t)
}

func (t *Tag) Resolved() bool {
	return t.Object != nil
}

func (t *Tag) Write() error {
	typ, err := objectType(t.Object)
	if err != nil {
		return err
	}
	b := new(bytes.Buffer)
	fmt.Fprintf(b, "object %s\ntype %s\ntag %s\ntagger %s\n\n", t.Object.SHA1(), typ, t.Name, t.Tagger)
	b.Write(t.Data)

	id, err := t.repo.writeObject("tag", b.Bytes())
	if err != nil {
		return err
	}
	t.id = id
	return t.repo.NewRef("refs/tags/"+t.Name, t.id).Write()
}

func (r *Repository) NewTag(name string, obj Object, tagger *User, msg string) *Tag {
	return &Tag{
		repo:   r,
		Object: obj,
		Name:   name,
		Tagger: tagger,
		Data:   []byte(msg),
	}
}
