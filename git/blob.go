package git

import "bytes"

type Blob struct {
	id   SHA1
	repo *Repository
	od   ObjectData
	Data []byte
}

func newBlob(id SHA1, repo *Repository) *Blob {
	return &Blob{
		id:   id,
		repo: repo,
	}
}

func (b *Blob) SHA1() SHA1 {
	return b.id
}

func (b *Blob) Parse(data []byte) error {
	b.Data = cloneBytes(data)
	return nil
}

func (b *Blob) Resolve() error {
	return b.repo.Resolve(b)
}

func (b *Blob) Resolved() bool {
	return b.Data != nil
}

func (b *Blob) Write() error {
	if len(b.Data) > 0 {
		b.od = bytes.NewReader(b.Data)
	}
	id, err := b.repo.writeObject("blob", b.od)
	if err == nil {
		b.id = id
	}
	return err
}

func cloneBytes(b []byte) []byte {
	n := len(b)
	dst := make([]byte, n, n)
	copy(dst, b)
	return dst
}

func (r *Repository) NewBlob(data ObjectData) *Blob {
	return &Blob{
		repo: r,
		od:   data,
	}
}
