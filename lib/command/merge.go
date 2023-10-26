package command

import (
	"bufio"
	"building-git/lib/command/write_commit"
	"building-git/lib/merge"
	"building-git/lib/repository"
	"io"
	"path/filepath"
	"time"
)

type MergeOption struct {
}

type Merge struct {
	rootPath string
	args     []string
	options  MergeOption
	repo     *repository.Repository
	inputs   *merge.Inputs
	stdin    io.Reader
	stdout   io.Writer
	stderr   io.Writer
}

func NewMerge(dir string, args []string, options MergeOption, stdin io.Reader, stdout, stderr io.Writer) (*Merge, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Merge{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdin:    stdin,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (m *Merge) Run() int {
	m.inputs, _ = merge.NewInputs(m.repo, repository.HEAD, m.args[0])
	m.resolveMerge()
	m.commitMerge()

	return 0
}

func (m *Merge) resolveMerge() {
	m.repo.Index.LoadForUpdate()

	merge := merge.NewResolve(m.repo, m.inputs)
	merge.Execute()

	m.repo.Index.WriteUpdates()
}

func (m *Merge) commitMerge() {
	parents := []string{m.inputs.LeftOid, m.inputs.RightOid}
	reader := bufio.NewReader(m.stdin)
	message, _ := reader.ReadString('\n')

	write_commit.WriteCommit(m.repo, parents, message, time.Now())

}
