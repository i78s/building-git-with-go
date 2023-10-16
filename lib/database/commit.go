package database

import (
	"bufio"
	"io"
	"strings"
	"time"
)

type Commit struct {
	parents []string
	oid     string
	tree    string
	author  *Author
	message string
}

func NewCommit(parents []string, tree string, author *Author, message string) *Commit {
	return &Commit{
		parents: parents,
		tree:    tree,
		author:  author,
		message: message,
	}
}

func ParseCommit(reader *bufio.Reader) (*Commit, error) {
	headers := make(map[string][]string)
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
		if len(parts) == 1 {
			parts = append(parts, "")
		}
		if headers[parts[0]] == nil {
			headers[parts[0]] = make([]string, 0)
		}
		headers[parts[0]] = append(headers[parts[0]], parts[1])
	}

	author, err := ParseAuthor(headers["author"][0])
	if err != nil {
		return nil, err
	}

	if headers["parent"] == nil {
		headers["parent"] = []string{""}
	}

	return NewCommit(
		headers["parent"],
		headers["tree"][0],
		author,
		message), nil
}

func (c *Commit) TitleLine() string {
	return strings.Split(c.message, "\n")[0]
}

func (c *Commit) Date() time.Time {
	return c.author.time
}

func (c *Commit) Type() string {
	return "commit"
}

func (c Commit) String() string {
	lines := []string{
		"tree " + c.tree,
	}

	for _, p := range c.parents {
		lines = append(lines, "parent "+p)
	}
	lines = append(lines,
		"author "+c.author.String(),
		"committer "+c.author.String(),
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

func (c *Commit) Parent() string {
	return c.parents[0]
}

func (c *Commit) Author() *Author {
	return c.author
}

func (c *Commit) Message() string {
	return c.message
}
