package database

import (
	"bufio"
	"io"
	"strings"
	"time"
)

type Commit struct {
	Parents   []string
	oid       string
	tree      string
	author    *Author
	committer *Author
	message   string
}

func NewCommit(parents []string, tree string, author, committer *Author, message string) *Commit {
	return &Commit{
		Parents:   parents,
		tree:      tree,
		author:    author,
		committer: committer,
		message:   message,
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
	committer, err := ParseAuthor(headers["committer"][0])
	if err != nil {
		return nil, err
	}

	if headers["parent"] == nil {
		headers["parent"] = []string{}
	}

	return NewCommit(
		headers["parent"],
		headers["tree"][0],
		author,
		committer,
		message), nil
}

func (c *Commit) IsMerge() bool {
	return len(c.Parents) > 1
}

func (c *Commit) TitleLine() string {
	return strings.Split(c.message, "\n")[0]
}

func (c *Commit) Date() time.Time {
	return c.committer.time
}

func (c *Commit) Type() string {
	return "commit"
}

func (c Commit) String() string {
	lines := []string{
		"tree " + c.tree,
	}

	for _, p := range c.Parents {
		lines = append(lines, "parent "+p)
	}
	lines = append(lines,
		"author "+c.author.String(),
		"committer "+c.committer.String(),
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
	if len(c.Parents) == 0 {
		return ""
	}
	return c.Parents[0]
}

func (c *Commit) Author() *Author {
	return c.author
}

func (c *Commit) Message() string {
	return c.message
}
