package database

import (
	"bufio"
	"strings"
)

type Commit struct {
	parent  string
	oid     string
	tree    string
	author  string
	message string
}

func NewCommit(parent, tree string, author string, message string) *Commit {
	return &Commit{
		parent:  parent,
		tree:    tree,
		author:  author,
		message: message,
	}
}

func ParseCommit(reader *bufio.Reader) (GitObject, error) {
	headers := make(map[string]string)

	for {
		line, _, err := reader.ReadLine()
		if err != nil {
			return nil, err
		}

		strLine := string(line)
		if strLine == "" {
			break
		}

		parts := strings.SplitN(strLine, " ", 2)
		headers[parts[0]] = parts[1]
	}

	rest, _ := reader.ReadString('\n')

	return NewCommit(
		headers["parent"],
		headers["tree"],
		headers["author"],
		rest), nil
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
		"author "+c.author,
		"committer "+c.author,
		"",
		c.message,
	)

	return strings.Join(lines, "\n")
}

func (c *Commit) Oid() string {
	return c.oid
}

func (c *Commit) SetOid(oid string) {
	c.oid = oid
}
