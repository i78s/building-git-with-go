package commit

import (
	"building-git/jit/author"
	"strings"
)

type Commit struct {
	oid     string
	tree    string
	author  *author.Author
	message string
}

func NewCommit(tree string, author *author.Author, message string) *Commit {
	return &Commit{
		tree:    tree,
		author:  author,
		message: message,
	}
}

func (c *Commit) Type() string {
	return "commit"
}

func (c Commit) String() string {
	lines := []string{
		"tree " + c.tree,
		"author " + c.author.String(),
		"committer " + c.author.String(),
		"",
		c.message,
	}

	return strings.Join(lines, "\n")
}

func (c *Commit) GetOid() string {
	return c.oid
}

func (c *Commit) SetOid(oid string) {
	c.oid = oid
}
