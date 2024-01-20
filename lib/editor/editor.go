package editor

import (
	"os"
	"os/exec"
	"strings"
)

const DEFAULT_EDITOR = "vi"

type Executable interface {
	Run() error
}

func EditorCmdFactory() func(path string) Executable {
	return func(path string) Executable {
		if editor := os.Getenv("GIT_EDITOR"); editor != "" {
			return exec.Command(editor, path)
		}
		if editor := os.Getenv("VISUAL"); editor != "" {
			return exec.Command(editor, path)
		}
		if editor := os.Getenv("EDITOR"); editor != "" {
			return exec.Command(editor, path)
		}
		return exec.Command(DEFAULT_EDITOR, path)
	}
}

type Editor struct {
	path    string
	command Executable
	closed  bool
	file    *os.File
}

func Edit(path string, command Executable, fn func(e *Editor)) string {
	editor := NewEditor(path, command)
	fn(editor)
	return editor.EditFile()
}

func EditFile(path string, command Executable, isTTY bool, fn func(e *Editor)) string {
	return Edit(path, command, func(e *Editor) {
		fn(e)
		if !isTTY {
			e.Close()
		}
	})
}

func NewEditor(path string, command Executable) *Editor {
	return &Editor{
		path:    path,
		command: command,
	}
}

func (e *Editor) Puts(s string) {
	if e.closed {
		return
	}
	_, _ = e.File().WriteString(s + "\n")
}

func (e *Editor) Note(s string) {
	if e.closed {
		return
	}
	for _, line := range strings.Split(s, "\n") {
		_, _ = e.File().WriteString("# " + line + "\n")
	}
}

func (e *Editor) Close() {
	e.closed = true
}

func (e *Editor) EditFile() string {
	if e.file != nil {
		_ = e.file.Close()
	}
	if !e.closed {
		if err := e.command.Run(); err != nil {
			panic("There was a problem with the editor.")
		}
	}

	content, err := os.ReadFile(e.path)
	if err != nil {
		panic(err)
	}

	return e.removeNotes(string(content))
}

func (e *Editor) removeNotes(s string) string {
	lines := strings.Split(s, "\n")
	var result []string
	for _, line := range lines {
		if !strings.HasPrefix(line, "#") &&
			strings.TrimSpace(line) != "" {
			result = append(result, line)
		}
	}
	return strings.Join(result, "\n") + "\n"
}

func (e *Editor) File() *os.File {
	if e.file == nil {
		file, err := os.OpenFile(e.path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		e.file = file
		return file
	}
	return e.file
}
