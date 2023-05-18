package commit

import (
	"building-git/jit/author"
	"strings"
)

type Commit struct {
	parent  string
	oid     string
	tree    string
	author  *author.Author
	message string
}

func NewCommit(parent, tree string, author *author.Author, message string) *Commit {
	return &Commit{
		parent:  parent,
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
	}

	if c.parent != "" {
		lines = append(lines, "parent "+c.parent)
	}
	lines = append(lines,
		"author "+c.author.String(),
		"committer "+c.author.String(),
		"",
		c.message,
	)

	return strings.Join(lines, "\n")
}

func (c *Commit) GetOid() string {
	return c.oid
}

func (c *Commit) SetOid(oid string) {
	c.oid = oid
}
