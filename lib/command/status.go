package command

import (
	"building-git/lib/repository"
	"building-git/lib/sortedmap"
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/fatih/color"
)

const LABEL_WIDTH = 12
const CONFLICT_LABEL_WIDTH = 17

var SHORT_STATUS = map[repository.ChangeType]string{
	repository.Added:    "A",
	repository.Deleted:  "D",
	repository.Modified: "M",
}

var LONG_STATUS = map[any]string{
	repository.Added:    "new file:   ",
	repository.Deleted:  "deleted:    ",
	repository.Modified: "modified:   ",
}

var CONFLICT_LONG_STATUS = map[any]string{
	"123": "both modified:",
	"12":  "deleted by them:",
	"13":  "deleted by us:",
	"23":  "both added:",
	"2":   "added by us:",
	"3":   "added by them:",
}

var CONFLICT_SHORT_STATUS = map[any]string{
	"123": "UU",
	"12":  "UD",
	"13":  "DU",
	"23":  "AA",
	"2":   "AU",
	"3":   "UA",
}

var UI_LABELS = map[string]map[any]string{
	"normal":   LONG_STATUS,
	"conflict": CONFLICT_LONG_STATUS,
}

var UI_WIDTHS = map[string]int{
	"normal":   LABEL_WIDTH,
	"conflict": CONFLICT_LABEL_WIDTH,
}

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
		return
	}
	s.printLongFormat()
}

func (s *Status) printLongFormat() {
	s.printBranchStatus()

	s.printChanges("Changes to be committed", *s.status.IndexChanges, color.New(color.FgGreen), "normal")
	s.printChanges("Unmerged paths", *s.status.Conflicts, color.New(color.FgRed), "conflict")
	s.printChanges("Changes not staged for commit", *s.status.WorkspaceChanges, color.New(color.FgRed), "normal")
	s.printChanges("Untracked files", *s.status.Untracked, color.New(color.FgRed), "normal")
	s.printCommitStatus()
}

func (s *Status) printBranchStatus() {
	current, _ := s.repo.Refs.CurrentRef("")

	if current.IsHead() {
		color.New(color.FgRed).Fprint(s.stdout, "Not currently on any branch.\n")
		return
	}
	short, _ := current.ShortName()
	fmt.Fprintf(s.stdout, "On branch %s\n", short)
}

func (s *Status) printChanges(message string, changeset interface{}, color *color.Color, labelSet string) {
	cset, ok := changeset.(sortedmap.SortedMap[any])
	if !ok {
		panic("")
	}
	if cset.Len() == 0 {
		return
	}

	labels := UI_LABELS[labelSet]
	width := UI_WIDTHS[labelSet]

	fmt.Fprintln(s.stdout, message)
	fmt.Fprintln(s.stdout)

	cset.Iterate(func(path string, change interface{}) {
		status := ""
		if ctype, ok := change.(repository.ChangeType); ok {
			status = labels[ctype]
		} else if statuses, ok := change.([]string); ok {
			sort.Strings(statuses)
			status = labels[strings.Join(statuses, "")]
		}

		if len(status) < width {
			status += strings.Repeat(" ", width-len(status))
		}
		color.Fprintf(s.stdout, "\t%s%s\n", status, path)
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
	if statuses, exists := s.status.Conflicts.Get(path); exists {
		sort.Strings(statuses)
		return CONFLICT_SHORT_STATUS[strings.Join(statuses, "")]
	}

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
