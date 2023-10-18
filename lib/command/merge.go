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
	headOid, _ := m.repo.Refs.ReadHead()
	revision := repository.NewRevision(m.repo, m.args[0])
	mergeOid, _ := revision.Resolve(repository.COMMIT)

	common := merge.NewBases(m.repo.Database, headOid, mergeOid)
	baseOid := common.Find()[0]

	m.repo.Index.LoadForUpdate()

	treeDiff := m.repo.Database.TreeDiff(baseOid, mergeOid, nil)
	migration := m.repo.Migration(treeDiff)
	migration.ApplyChanges()

	m.repo.Index.WriteUpdates()

	reader := bufio.NewReader(m.stdin)
	message, _ := reader.ReadString('\n')

	write_commit.WriteCommit(m.repo, []string{headOid, mergeOid}, message, time.Now())

	return 0
}
