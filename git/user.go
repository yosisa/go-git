package git

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"time"
)

var ErrUnknownFormat = errors.New("Unknown format")

type User struct {
	Name  string
	Email string
	Date  time.Time
}

func newUser(data []byte) (*User, error) {
	var (
		user User
		pos  int
	)
	if pos = bytes.IndexByte(data, '<'); pos == -1 {
		return nil, ErrUnknownFormat
	}
	user.Name = string(data[:pos-1])
	data = data[pos+1:]

	if pos = bytes.IndexByte(data, '>'); pos == -1 {
		return nil, ErrUnknownFormat
	}
	user.Email = string(data[:pos])
	data = data[pos+2:]

	if pos = bytes.IndexByte(data, ' '); pos == -1 {
		return nil, ErrUnknownFormat
	}
	sec, err := strconv.ParseInt(string(data[:pos]), 10, 64)
	if err != nil {
		return nil, err
	}
	t, err := time.Parse("-0700", string(data[pos+1:]))
	if err != nil {
		return nil, err
	}
	user.Date = time.Unix(sec, 0).In(t.Location())
	return &user, nil
}

func NewUser(name, email string) *User {
	return &User{
		Name:  name,
		Email: email,
		Date:  time.Now(),
	}
}

func (u *User) String() string {
	return fmt.Sprintf("%s <%s> %d %s", u.Name, u.Email, u.Date.Unix(), u.Date.Format("-0700"))
}
