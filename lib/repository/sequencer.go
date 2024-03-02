package repository

import (
	"bufio"
	"building-git/lib/database"
	"building-git/lib/lockfile"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

type Sequencer struct {
	repo     *Repository
	pathname string
	todoPath string
	todoFile *lockfile.Lockfile
	commands []*database.Commit
}

func NewSequencer(repo *Repository) *Sequencer {
	pathname := filepath.Join(repo.GitPath, "sequencer")
	return &Sequencer{
		repo:     repo,
		pathname: pathname,
		todoPath: filepath.Join(pathname, "todo"),
		commands: []*database.Commit{},
	}
}

func (s *Sequencer) Start() {
	os.Mkdir(s.pathname, os.ModePerm)
	s.openTodoFile()
}

func (s *Sequencer) Pick(commit *database.Commit) {
	s.commands = append(s.commands, commit)
}

func (s *Sequencer) NextCommand() *database.Commit {
	if len(s.commands) == 0 {
		return nil
	}
	return s.commands[0]
}

func (s *Sequencer) DropCommand() {
	s.commands = s.commands[1:]
}

func (s *Sequencer) Load() {
	s.openTodoFile()
	fileInfo, err := os.Stat(s.todoPath)
	if err != nil {
		return
	}
	if !fileInfo.Mode().IsRegular() {
		return
	}

	file, err := os.Open(s.todoPath)
	if err != nil {
		fmt.Printf("Error opening file: %v\n", err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	regex := regexp.MustCompile(`^pick (\S+) (.*)$`)

	for scanner.Scan() {
		line := scanner.Text()
		matches := regex.FindStringSubmatch(line)
		if matches != nil {
			oid := matches[1]
			oids, _ := s.repo.Database.PrefixMatch(oid)
			obj, _ := s.repo.Database.Load(oids[0])
			s.commands = append(s.commands, obj.(*database.Commit))
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("Error reading file: %v\n", err)
	}
}

func (s *Sequencer) Dump() {
	if s.todoFile == nil {
		return
	}

	for _, commit := range s.commands {
		short := s.repo.Database.ShortOid(commit.Oid())
		s.todoFile.Write([]byte(fmt.Sprintf("pick %s %s\n", short, commit.TitleLine())))
	}
	s.todoFile.Commit()
	fmt.Println()
}

func (s *Sequencer) Quit() {
	os.RemoveAll(s.pathname)
}

func (s *Sequencer) openTodoFile() {
	fileInfo, err := os.Stat(s.pathname)
	if err != nil {
		return
	}
	if !fileInfo.IsDir() {
		return
	}

	s.todoFile = lockfile.NewLockfile(s.todoPath)
	s.todoFile.HoldForUpdate()
}
