package database

import (
	"bufio"
	"io"
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

func ParseCommit(reader *bufio.Reader) (*Commit, error) {
	headers := make(map[string]string)
	message := ""

	for {
		line, err := reader.ReadString('\n')
		line = strings.TrimSpace(string(line))
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if line == "" {
			messageBytes, err := reader.ReadBytes('\x00')
			if err != nil && err != io.EOF {
				return nil, err
			}
			message = string(messageBytes)
			break
		}

		parts := strings.SplitN(line, " ", 2)

		headers[parts[0]] = parts[1]
	}

	return NewCommit(
		headers["parent"],
		headers["tree"],
		headers["author"],
		message), nil
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

func (c *Commit) Tree() string {
	return c.tree
}
