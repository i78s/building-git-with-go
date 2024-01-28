package command

import (
	"building-git/lib/command/write_commit"
	"building-git/lib/editor"
	"building-git/lib/merge"
	"building-git/lib/repository"
	"fmt"
	"io"
	"path/filepath"
	"time"
)

type MergeMode string

const (
	Run      MergeMode = "run"
	Continue MergeMode = "continue"
	Abort    MergeMode = "abort"
)

type MergeOption struct {
	write_commit.ReadOption
	Mode      MergeMode
	Edit      bool
	IsTTY     bool
	EditorCmd func(path string) editor.Executable
}

type Merge struct {
	rootPath    string
	args        []string
	options     MergeOption
	repo        *repository.Repository
	writeCommit *write_commit.WriteCommit
	inputs      *merge.Inputs
	stdout      io.Writer
	stderr      io.Writer
}

func NewMerge(dir string, args []string, options MergeOption, stdout, stderr io.Writer) (*Merge, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)
	writeCommit := write_commit.NewWriteCommit(repo, options.EditorCmd)

	return &Merge{
		rootPath:    rootPath,
		args:        args,
		options:     options,
		repo:        repo,
		writeCommit: writeCommit,
		stdout:      stdout,
		stderr:      stderr,
	}, nil
}

func (m *Merge) Run() int {
	if m.options.Mode == Abort {
		if err := m.handleAbort(); err != nil {
			fmt.Fprintf(m.stderr, "fatal: %s\n", err.Error())
			return 128
		}
		return 0
	}
	if m.options.Mode == Continue {
		if err := m.handleContinue(); err != nil {
			if _, ok := err.(*repository.PendingCommitError); ok {
				fmt.Fprintf(m.stderr, "fatal: %s\n", err.Error())
			} else {
				fmt.Fprintf(m.stderr, "%s\n", err.Error())
			}
			return 128
		}
		return 0
	}

	if m.writeCommit.PendingCommit().InProgress() {
		if err := m.handleInProgressMerge(); err != nil {
			fmt.Fprintf(m.stderr, "%s\n", err.Error())
			return 128
		}
	}

	inputs, _ := merge.NewInputs(m.repo, repository.HEAD, m.args[0])
	m.inputs = inputs
	m.repo.Refs.UpateRef(repository.ORIG_HEAD, inputs.LeftOid())

	if m.inputs.IsAlreadyMerged() {
		m.handleMergedAncestor()
		return 0
	}
	if m.inputs.IsFastForward() {
		m.handleFastForward()
		return 0
	}

	err := m.writeCommit.PendingCommit().Start(m.inputs.RightOid())
	if err != nil {
		return 1
	}
	if err := m.resolveMerge(); err != nil {
		fmt.Fprintf(m.stdout, "Automatic merge failed; fix conflicts and then commit the result.")
		return 1
	}
	m.commitMerge()

	return 0
}

func (m *Merge) resolveMerge() error {
	m.repo.Index.LoadForUpdate()

	merge := merge.NewResolve(
		m.repo,
		m.inputs,
		func(fn func() string) {
			info := fn()
			fmt.Fprintf(m.stdout, info+"\n")
		},
	)
	merge.Execute()

	m.repo.Index.WriteUpdates()
	if m.repo.Index.IsConflict() {
		if err := m.failOnConflict(); err != nil {
			return err
		}
	}
	return nil
}

func (m *Merge) failOnConflict() error {
	path := m.writeCommit.PendingCommit().MessagePath
	editor.EditFile(path, m.options.EditorCmd(path), m.options.IsTTY, func(e *editor.Editor) {
		message, err := m.writeCommit.ReadMessage(m.options.ReadOption)
		if err != nil {
			message = m.defaultCommitMessage()
		}
		e.Puts(message)
		e.Puts("")
		e.Note("Conflicts:")
		for name := range m.repo.Index.ConflictPaths() {
			e.Note("\t" + name)
		}
		e.Close()
	})
	return fmt.Errorf("detect conflict")
}

func (m *Merge) commitMerge() {
	parents := []string{m.inputs.LeftOid(), m.inputs.RightOid()}
	message := m.composeMessage()

	m.writeCommit.WriteCommit(parents, message, time.Now())
	m.writeCommit.PendingCommit().Clear()
}

func (m *Merge) composeMessage() string {
	path := m.writeCommit.PendingCommit().MessagePath
	return editor.EditFile(path, m.options.EditorCmd(path), m.options.IsTTY, func(e *editor.Editor) {
		message, err := m.writeCommit.ReadMessage(m.options.ReadOption)
		if err != nil {
			message = m.defaultCommitMessage()
		}
		e.Puts(message)
		e.Puts("")
		e.Note(write_commit.COMMIT_NOTES)

		if !m.options.Edit {
			e.Close()
		}
	})
}

func (m *Merge) defaultCommitMessage() string {
	return fmt.Sprintf("Merge commit %s", m.inputs.RightName())
}

func (m *Merge) handleMergedAncestor() {
	fmt.Fprintf(m.stdout, "Already up to date.\n")
}

func (m *Merge) handleFastForward() {
	a := m.repo.Database.ShortOid(m.inputs.LeftOid())
	b := m.repo.Database.ShortOid(m.inputs.RightOid())

	fmt.Fprintf(m.stdout, "Updating %s..%s\n", a, b)
	fmt.Fprintf(m.stdout, "Fast-forward\n")

	m.repo.Index.LoadForUpdate()
	treeDiff := m.repo.Database.TreeDiff(m.inputs.LeftOid(), m.inputs.RightOid(), nil)
	m.repo.Migration(treeDiff).ApplyChanges()

	m.repo.Index.WriteUpdates()
	m.repo.Refs.UpdateHead(m.inputs.RightOid())
}

func (m *Merge) handleAbort() error {
	err := m.repo.PendingCommit.Clear()

	m.repo.Index.LoadForUpdate()
	head, _ := m.repo.Refs.ReadHead()
	m.repo.HardReset(head)
	m.repo.Index.WriteUpdates()

	if err != nil {
		return err
	}
	return nil
}

func (m *Merge) handleContinue() error {
	m.repo.Index.Load()
	err := m.writeCommit.ResumeMerge(m.options.IsTTY)

	if err != nil {
		return err
	}
	return nil
}

func (m *Merge) handleInProgressMerge() error {
	message := "Merging is not possible because you have unmerged files."
	return fmt.Errorf(`error: %s
%s`, message, write_commit.CONFLICT_MESSAGE)
}
