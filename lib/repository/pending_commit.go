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

type PendingCommit struct {
	headPath    string
	MessagePath string
}

func NewPendingCommit(pathname string) *PendingCommit {
	return &PendingCommit{
		headPath:    filepath.Join(pathname, "MERGE_HEAD"),
		MessagePath: filepath.Join(pathname, "MERGE_MSG"),
	}
}

func (pc *PendingCommit) Start(oid string) error {
	if err := os.WriteFile(pc.headPath, []byte(oid+"\n"), 0666); err != nil {
		return err
	}
	return nil
}

func (pc *PendingCommit) InProgress() bool {
	_, err := os.Stat(pc.headPath)
	return !os.IsNotExist(err)
}

func (pc *PendingCommit) MergeOID() (string, error) {
	data, err := os.ReadFile(pc.headPath)
	if err != nil {
		return "", &PendingCommitError{fmt.Sprintf("There is no merge in progress (%s missing).", filepath.Base(pc.headPath))}
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

func (pc *PendingCommit) Clear() error {
	if err := os.Remove(pc.headPath); err != nil {
		return &PendingCommitError{fmt.Sprintf("There is no merge to abort (%s missing).", filepath.Base(pc.headPath))}
	}
	return os.Remove(pc.MessagePath)
}
