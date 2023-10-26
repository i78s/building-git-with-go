package repository

import (
	"building-git/lib/database"
	"fmt"
	"regexp"
	"sort"
	"strconv"
)

var (
	INVALID_NAME = regexp.MustCompile(`^\.|\/\.|\.\.|^\/|\/$|\.lock$|@\{|[\x00-\x20*:?\[\\^~\x7f]`)
	PARENT       = regexp.MustCompile(`^(.+)\^(\d*)$`)
	ANCESTOR     = regexp.MustCompile(`^(.+)~(\d+)$`)
	REF_ALIASES  = map[string]string{
		"@": HEAD,
	}
	COMMIT = "commit"
)

type InvalidObjectError struct {
	expr string
}

func (e *InvalidObjectError) Error() string {
	return fmt.Sprintf("Not a valid object name: '%s'.", e.expr)
}

type HintedError struct {
	Message string
	Hint    []string
}

type Context interface {
	ReadRef(name string) (string, error)
	CommitParent(oid string) (string, error)
}

type ParsedRevision interface {
	resolve(context *Revision) (string, error)
}

type Revision struct {
	repo   *Repository
	expr   string
	query  ParsedRevision
	Errors []HintedError
}

func IsValidRef(revision string) bool {
	return !INVALID_NAME.MatchString(revision)
}

func NewRevision(repo *Repository, expr string) *Revision {
	return &Revision{
		repo:   repo,
		expr:   expr,
		query:  parse(expr),
		Errors: []HintedError{},
	}
}

func (r *Revision) Resolve(otype string) (string, error) {
	invalidObjErr := &InvalidObjectError{r.expr}
	if r.query == nil {
		return "", invalidObjErr
	}

	oid, _ := r.query.resolve(r)
	if otype != "" {
		if _, err := r.loadTypedObject(oid, otype); err != nil {
			oid = ""
		}
	}

	if oid != "" {
		return oid, nil
	}
	return "", invalidObjErr
}

func (r *Revision) readRef(name string) (string, error) {
	oid, _ := r.repo.Refs.ReadRef(name)
	if oid != "" {
		return oid, nil
	}

	candidates, err := r.repo.Database.PrefixMatch(name)
	if err != nil {
		return "", err
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) > 1 {
		r.logAmbiguousSha1(name, candidates)
	}
	return "", nil
}

func (r *Revision) commitParent(oid string, n int) (string, error) {
	if n == 0 {
		n = 1
	}
	if oid == "" {
		return "", nil
	}
	commit, err := r.loadTypedObject(oid, COMMIT)
	if err != nil || commit == nil {
		return "", err
	}

	if c, ok := commit.(*database.Commit); ok {
		if len(c.Parents) > n-1 {
			return c.Parents[n-1], nil
		}
	}
	return "", nil
}

func (r *Revision) loadTypedObject(oid, otype string) (database.GitObject, error) {
	if oid == "" {
		return nil, fmt.Errorf("oid is empty")
	}

	obj, err := r.repo.Database.Load(oid)

	if err != nil {
		return nil, err
	}
	if obj.Type() == COMMIT {
		return obj, nil
	}

	message := fmt.Sprintf("object %s is a %s, not a %s", oid, obj.Type(), otype)
	r.Errors = append(r.Errors, HintedError{message, []string{}})

	return nil, fmt.Errorf(message)
}

func (r *Revision) logAmbiguousSha1(name string, candidates []string) error {
	sort.Strings(candidates)

	var objects []string
	for _, oid := range candidates {
		object, err := r.repo.Database.Load(oid)
		if err != nil {
			return err
		}

		short := r.repo.Database.ShortOid(object.Oid())
		info := fmt.Sprintf("  %s %s", short, object.Type())

		if commit, ok := object.(*database.Commit); ok {
			info = fmt.Sprintf("%s %s - %s", info, commit.Author().ShortDate(), commit.TitleLine())
		}

		objects = append(objects, info)
	}

	message := fmt.Sprintf("short SHA1 %s is ambiguous", name)
	hint := append([]string{"The candidates are:"}, objects...)
	r.Errors = append(r.Errors, HintedError{message, hint})

	return nil
}

func parse(revision string) ParsedRevision {
	if match := PARENT.FindStringSubmatch(revision); match != nil {
		rev := parse(match[1])
		n := 1
		if match[2] != "" {
			n, _ = strconv.Atoi(match[2])
		}
		if rev != nil {
			return &Parent{rev, n}
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
		return &ref{name}
	}
	return nil
}

type ref struct {
	name string
}

func (r *ref) resolve(context *Revision) (string, error) {
	return context.readRef(r.name)
}

type Parent struct {
	rev ParsedRevision
	n   int
}

func (p *Parent) resolve(context *Revision) (string, error) {
	oid, _ := p.rev.resolve(context)
	return context.commitParent(oid, p.n)
}

type Ancestor struct {
	rev ParsedRevision
	n   int
}

func (a *Ancestor) resolve(context *Revision) (string, error) {
	oid, _ := a.rev.resolve(context)
	for i := 0; i < a.n; i++ {
		var err error
		oid, err = context.commitParent(oid, 1)
		if err != nil {
			return "", err
		}
	}
	return oid, nil
}
