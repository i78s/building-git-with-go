package database

import (
	"fmt"
	"time"
)

type Author struct {
	name  string
	email string
	time  time.Time
}

func NewAuthor(name, email string, time time.Time) *Author {
	return &Author{
		name:  name,
		email: email,
		time:  time,
	}
}

func (a Author) String() string {
	timestamp := fmt.Sprintf("%d %s", a.time.Unix(), a.time.Format("-0700"))
	return fmt.Sprintf("%s <%s> %s", a.name, a.email, timestamp)
}
