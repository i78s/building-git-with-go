package write_commit

import (
	"building-git/lib/database"
	"building-git/lib/editor"
	"building-git/lib/repository"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"
)

const COMMIT_NOTES = `Please enter a commit message to explain why this merge is necessary,
especially if it merges an updated upstream into a topic branch.
Lines starting with '#' will be ignored, and an empty message aborts
the commit.`

const CONFLICT_MESSAGE = `hint: Fix them up in the work tree, and then use 'jit add/rm <file>'
hint: as appropriate to mark resolution and make a commit.
fatal: Exiting because of an unresolved conflict.`

const MERGE_NOTES = `

It looks like you may be committing a merge.
If this is not correct, please remove the file
\t.git/MERGE_HEAD
and try again.`

type WriteCommit struct {
	repo      *repository.Repository
	editorCmd func(path string) editor.Executable
}

func NewWriteCommit(
	repo *repository.Repository,
	editorCmd func(path string) editor.Executable,
) *WriteCommit {
	return &WriteCommit{
		repo:      repo,
		editorCmd: editorCmd,
	}
}

type ReadOption struct {
	Message string
	File    string
}

func (wc *WriteCommit) ReadMessage(option ReadOption) (string, error) {
	if option.Message != "" {
		return fmt.Sprintf("%s\n", option.Message), nil
	} else if option.File != "" {
		path, err := filepath.Abs(option.File)
		if err != nil {
			return "", fmt.Errorf("")
		}
		msg, err := os.ReadFile(path)
		return string(msg), err
	}
	return "", fmt.Errorf("")
}

func (wc *WriteCommit) WriteCommit(parents []string, message string, now time.Time) (*database.Commit, error) {
	if message == "" {
		return nil, fmt.Errorf("Aborting commit due to empty commit message.\n")
	}

	tree := wc.WriteTree()
	name, exists := os.LookupEnv("GIT_AUTHOR_NAME")
	if !exists {
		log.Fatalf("GIT_AUTHOR_NAME is not set")
	}
	email, exists := os.LookupEnv("GIT_AUTHOR_EMAIL")
	if !exists {
		log.Fatalf("GIT_AUTHOR_EMAIL is not set")
	}

	author := database.NewAuthor(name, email, now)

	commit := database.NewCommit(parents, tree.Oid(), author, message)
	wc.repo.Database.Store(commit)
	wc.repo.Refs.UpdateHead(commit.Oid())

	return commit, nil
}

func (wc *WriteCommit) WriteTree() *database.Tree {
	root := database.BuildTree(wc.repo.Index.EachEntry())
	root.Traverse(func(t database.TreeObject) {
		if gitObj, ok := t.(database.GitObject); ok {
			if err := wc.repo.Database.Store(gitObj); err != nil {
				log.Fatalf("Failed to store object: %v", err)
			}
		} else {
			log.Fatalf("Object does not implement GitObject interface")
		}
	})
	return root
}

func (wc *WriteCommit) PrintCommit(commit *database.Commit, w io.Writer) {
	ref, _ := wc.repo.Refs.CurrentRef("")
	info, _ := ref.ShortName()
	if ref.IsHead() {
		info = "detached HEAD"
	}
	oid := wc.repo.Database.ShortOid(commit.Oid())

	if commit.Parent() == "" {
		info += " (root-commit)"
	}
	info += fmt.Sprintf(" %s", oid)

	fmt.Fprintf(w, "[%s] %s\n", info, commit.TitleLine())
}

func (wc *WriteCommit) PendingCommit() *repository.PendingCommit {
	return wc.repo.PendingCommit
}

func (wc *WriteCommit) ResumeMerge(isTTY bool) error {
	err := wc.HandleConflictedIndex()
	if err != nil {
		return err
	}
	head, _ := wc.repo.Refs.ReadHead()
	mergeOid, err := wc.PendingCommit().MergeOID()
	if err != nil {
		return err
	}
	parants := []string{head, mergeOid}

	message := wc.composeMergeMessage(MERGE_NOTES, isTTY)
	wc.WriteCommit(parants, message, time.Now())
	wc.PendingCommit().Clear()
	return nil
}

func (wc *WriteCommit) composeMergeMessage(notes string, isTTY bool) string {
	path := wc.CommitMessagePath()
	return editor.EditFile(path, wc.editorCmd(path), isTTY, func(e *editor.Editor) {
		message, _ := wc.PendingCommit().MergeMessage()

		e.Puts(message)
		if notes != "" {
			e.Note(notes)
		}
		e.Puts("")
		e.Note(COMMIT_NOTES)
	})
}

func (wc *WriteCommit) CommitMessagePath() string {
	return filepath.Join(wc.repo.GitPath, "COMMIT_EDITMSG")
}

func (wc *WriteCommit) HandleConflictedIndex() error {
	if !wc.repo.Index.IsConflict() {
		return nil
	}
	message := "Committing is not possible because you have unmerged files."
	return fmt.Errorf(`error: %s
%s`, message, CONFLICT_MESSAGE)
}
