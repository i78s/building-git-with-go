package repository

import (
	"fmt"
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

type InvalidObjectError struct {
	expr string
}

func (e *InvalidObjectError) Error() string {
	return fmt.Sprintf("Not a valid object name: '%s'.", e.expr)
}

type Context interface {
	ReadRef(name string) (string, error)
	CommitParent(oid string) (string, error)
}

type ParsedRevision interface {
	resolve(context *Revision) (string, error)
}

type Revision struct {
	repo  *Repository
	expr  string
	query ParsedRevision
}

func IsValidRef(revision string) bool {
	return !INVALID_NAME.MatchString(revision)
}

func NewRevision(repo *Repository, expr string) *Revision {
	return &Revision{
		repo:  repo,
		expr:  expr,
		query: parse(expr),
	}
}

func (r *Revision) Resolve() (string, error) {
	err := &InvalidObjectError{r.expr}
	if r.query == nil {
		return "", err
	}

	oid, _ := r.query.resolve(r)
	if oid != "" {
		return oid, nil
	}
	return "", err
}

func (r *Revision) readRef(name string) (string, error) {
	return r.repo.Refs.ReadRef(name)
}

type CommitObject interface {
	Parent() string
}

func (r *Revision) commitParent(oid string) (string, error) {
	if oid == "" {
		return "", nil
	}
	commit, err := r.repo.Database.Load(oid)
	if err != nil {
		return "", err
	}

	if c, ok := commit.(CommitObject); ok {
		return c.Parent(), nil
	}
	return "", nil
}

func parse(revision string) ParsedRevision {
	if match := PARENT.FindStringSubmatch(revision); match != nil {
		rev := parse(match[1])
		if rev != nil {
			return &Parent{rev}
		}
	} else if match := ANCESTOR.FindStringSubmatch(revision); match != nil {
		rev := parse(match[1])
		if rev != nil {
			n, _ := strconv.Atoi(match[2])
			return &Ancestor{rev, n}
		}
	} else if IsValidRef(revision) {
		name := REF_ALIASES[revision]
		if name == "" {
			name = revision
		}
		return &Ref{name}
	}
	return nil
}

type Ref struct {
	name string
}

func (r *Ref) resolve(context *Revision) (string, error) {
	return context.readRef(r.name)
}

type Parent struct {
	rev ParsedRevision
}

func (p *Parent) resolve(context *Revision) (string, error) {
	oid, _ := p.rev.resolve(context)
	return context.commitParent(oid)
}

type Ancestor struct {
	rev ParsedRevision
	n   int
}

func (a *Ancestor) resolve(context *Revision) (string, error) {
	oid, _ := a.rev.resolve(context)
	for i := 0; i < a.n; i++ {
		var err error
		oid, err = context.commitParent(oid)
		if err != nil {
			return "", err
		}
	}
	return oid, nil
}
