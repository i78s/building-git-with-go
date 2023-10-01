package database

import (
	"fmt"
	"strings"
	"time"
)

const (
	timeFormat = "2006-01-02T15:04:05 -0700"
)

type Author struct {
	Name  string
	Email string
	time  time.Time
}

func NewAuthor(name, email string, time time.Time) *Author {
	return &Author{
		Name:  name,
		Email: email,
		time:  time,
	}
}

func ParseAuthor(s string) (*Author, error) {
	parts := strings.Split(s, "<")
	name := strings.TrimSpace(parts[0])

	parts = strings.Split(parts[1], ">")
	email := strings.TrimSpace(parts[0])

	t, err := time.Parse(timeFormat, strings.TrimSpace(parts[1]))
	if err != nil {
		return nil, err
	}

	return &Author{name, email, t}, nil
}

func (a *Author) ShortDate() string {
	return a.time.Format("2006-01-02")
}

func (a *Author) ReadableTime() string {
	return a.time.Format("Mon Jan 2 15:04:05 2006 -0700")
}

func (a *Author) String() string {
	return fmt.Sprintf("%s <%s> %s", a.Name, a.Email, a.time.Format(timeFormat))
}
