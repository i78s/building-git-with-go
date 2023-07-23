package command

import (
	"building-git/lib/repository"
	"building-git/lib/sortedmap"
	"fmt"
	"io"
	"path/filepath"

	"github.com/fatih/color"
)

var SHORT_STATUS = map[repository.ChangeType]string{
	repository.Added:    "A",
	repository.Deleted:  "D",
	repository.Modified: "M",
}

var LONG_STATUS = map[repository.ChangeType]string{
	repository.Added:    "new file:   ",
	repository.Deleted:  "deleted:    ",
	repository.Modified: "modified:   ",
}

const LABEL_WIDTH = 12

type StatusOption struct {
	Porcelain bool
}

type Status struct {
	rootPath string
	args     []string
	options  StatusOption
	repo     *repository.Repository
	status   *repository.Status
	stdout   io.Writer
	stderr   io.Writer
}

func NewStatus(dir string, args []string, options StatusOption, stdout, stderr io.Writer) (*Status, error) {
	rootPath, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	repo := repository.NewRepository(rootPath)

	return &Status{
		rootPath: rootPath,
		args:     args,
		options:  options,
		repo:     repo,
		stdout:   stdout,
		stderr:   stderr,
	}, nil
}

func (s *Status) Run() int {
	s.repo.Index.LoadForUpdate()

	status, err := s.repo.Status()
	s.status = status
	if err != nil {
		fmt.Fprintf(s.stderr, "fatal: %v", err)
		return 128
	}

	s.repo.Index.WriteUpdates()
	s.printResults()

	return 0
}

func (s *Status) printResults() {
	if s.options.Porcelain {
		s.printPorcelainFormat()

		s.status.IndexChanges.Iterate(func(path string, change repository.ChangeType) {
			fmt.Println(path, change)
		})
		return
	}
	s.printLongFormat()
}

func (s *Status) printLongFormat() {
	s.printChanges("Changes to be committed", *s.status.IndexChanges, color.New(color.FgGreen))
	s.printChanges("Changes not staged for commit", *s.status.WorkspaceChanges, color.New(color.FgRed))
	s.printUntrackedChanges("Untracked files", *s.status.Untracked, color.New(color.FgRed))

	s.printCommitStatus()
}

func (s *Status) printChanges(message string, changeset sortedmap.SortedMap[repository.ChangeType], color *color.Color) {
	if changeset.Len() == 0 {
		return
	}

	fmt.Fprintln(s.stdout, message)
	fmt.Fprintln(s.stdout)

	changeset.Iterate(func(path string, change repository.ChangeType) {
		status := LONG_STATUS[change]
		color.Fprintf(s.stdout, "\t%s%s\n", status, path)
	})

	fmt.Fprintln(s.stdout)
}

func (s *Status) printUntrackedChanges(message string, changeset sortedmap.SortedMap[struct{}], color *color.Color) {
	if changeset.Len() == 0 {
		return
	}

	fmt.Fprintln(s.stdout, message)
	fmt.Fprintln(s.stdout)

	changeset.Iterate(func(path string, _ struct{}) {
		color.Fprintf(s.stdout, "\t%s\n", path)
	})

	fmt.Fprintln(s.stdout)
}

func (s *Status) printCommitStatus() {
	if s.status.IndexChanges.Len() > 0 {
		return
	}

	if s.status.WorkspaceChanges.Len() > 0 {
		fmt.Fprintln(s.stdout, "no changes added to commit")
	} else if s.status.Untracked.Len() > 0 {
		fmt.Fprintln(s.stdout, "nothing added to commit but untracked files present")
	} else {
		fmt.Fprintln(s.stdout, "nothing to commit, working tree clean")
	}
}

func (s *Status) printPorcelainFormat() {
	s.status.Changed.Iterate(func(filename string, _ struct{}) {
		status := s.statusFor(filename)
		fmt.Fprintf(s.stdout, "%s %s\n", status, filename)
	})

	s.status.Untracked.Iterate(func(filename string, _ struct{}) {
		fmt.Fprintf(s.stdout, "?? %s\n", filename)
	})
}

func (s *Status) statusFor(path string) string {
	left := " "
	if ctype, exists := s.status.IndexChanges.Get(path); exists {
		if status, exists := SHORT_STATUS[ctype]; exists {
			left = status
		}
	}

	right := " "
	if ctype, exists := s.status.WorkspaceChanges.Get(path); exists {
		if status, exists := SHORT_STATUS[ctype]; exists {
			right = status
		}
	}

	return left + right
}
