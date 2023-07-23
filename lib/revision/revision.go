package revision

import (
	"regexp"
	"strconv"
)

var (
	INVALID_NAME = regexp.MustCompile(`^\.|\/\.|\.\.|^\/|\/$|\.lock$|@\{|[\x00-\x20*:?\[\\^~\x7f]`)
	PARENT       = regexp.MustCompile(`^(.+)\^$`)
	ANCESTOR     = regexp.MustCompile(`^(.+)~(\d+)$`)
	REF_ALIASES  = map[string]string{
		"@": "HEAD",
	}
)

type Revision interface{}

func Parse(revision string) Revision {
	if match := PARENT.FindStringSubmatch(revision); match != nil {
		rev := Parse(match[1])
		if rev != nil {
			return Parent{rev}
		}
	} else if match := ANCESTOR.FindStringSubmatch(revision); match != nil {
		rev := Parse(match[1])
		if rev != nil {
			n, _ := strconv.Atoi(match[2])
			return Ancestor{rev, n}
		}
	} else if IsValidRef(revision) {
		name := REF_ALIASES[revision]
		if name == "" {
			name = revision
		}
		return Ref{name}
	}
	return nil
}

func IsValidRef(revision string) bool {
	return !INVALID_NAME.MatchString(revision)
}

type Ref struct {
	name string
}

type Parent struct {
	rev Revision
}

type Ancestor struct {
	rev Revision
	n   int
}
