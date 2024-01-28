package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PendingCommitError struct {
	Message string
}

func (e *PendingCommitError) Error() string {
	return e.Message
}

type MergeType int

const (
	Merge MergeType = iota
	CherryPick
)

var HEAD_FILES = map[MergeType]string{
	Merge:      "MERGE_HEAD",
	CherryPick: "CHERRY_PICK_HEAD",
}

type PendingCommit struct {
	pathname    string
	MessagePath string
}

func NewPendingCommit(pathname string) *PendingCommit {
	return &PendingCommit{
		pathname:    pathname,
		MessagePath: filepath.Join(pathname, "MERGE_MSG"),
	}
}

func (pc *PendingCommit) Start(oid string, mtype MergeType) error {
	path := filepath.Join(pc.pathname, HEAD_FILES[mtype])
	if err := os.WriteFile(path, []byte(oid+"\n"), 0666); err != nil {
		return err
	}
	return nil
}

func (pc *PendingCommit) InProgress() bool {
	return pc.MergeType() != -1
}

func (pc *PendingCommit) MergeType() MergeType {
	for mtype, name := range HEAD_FILES {
		path := filepath.Join(pc.pathname, name)
		_, err := os.Stat(path)
		if !os.IsNotExist(err) {
			return mtype
		}
	}
	return -1
}

func (pc *PendingCommit) MergeOID(mtype MergeType) (string, error) {
	headPath := filepath.Join(pc.pathname, HEAD_FILES[mtype])
	data, err := os.ReadFile(headPath)
	if err != nil {
		return "", &PendingCommitError{fmt.Sprintf("There is no merge in progress (%s missing).", filepath.Base(headPath))}
	}
	return strings.TrimSpace(string(data)), nil
}

func (pc *PendingCommit) MergeMessage() (string, error) {
	data, err := os.ReadFile(pc.MessagePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (pc *PendingCommit) Clear(mtype MergeType) error {
	headPath := filepath.Join(pc.pathname, HEAD_FILES[mtype])
	if err := os.Remove(headPath); err != nil {
		return &PendingCommitError{fmt.Sprintf("There is no merge to abort (%s missing).", filepath.Base(headPath))}
	}
	return os.Remove(pc.MessagePath)
}
