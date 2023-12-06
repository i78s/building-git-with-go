package repository

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type PendingCommit struct {
	headPath    string
	messagePath string
}

func NewPendingCommit(pathname string) *PendingCommit {
	return &PendingCommit{
		headPath:    filepath.Join(pathname, "MERGE_HEAD"),
		messagePath: filepath.Join(pathname, "MERGE_MSG"),
	}
}

func (pc *PendingCommit) Start(oid, message string) error {
	if err := os.WriteFile(pc.headPath, []byte(oid+"\n"), 0666); err != nil {
		return err
	}
	return os.WriteFile(pc.messagePath, []byte(message), 0666)
}

func (pc *PendingCommit) InProgress() bool {
	_, err := os.Stat(pc.headPath)
	return !os.IsNotExist(err)
}

func (pc *PendingCommit) MergeOID() (string, error) {
	data, err := os.ReadFile(pc.headPath)
	if err != nil {
		return "", fmt.Errorf("there is no merge in progress (%s missing): %w", filepath.Base(pc.headPath), err)
	}
	return strings.TrimSpace(string(data)), nil
}

func (pc *PendingCommit) MergeMessage() (string, error) {
	data, err := os.ReadFile(pc.messagePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (pc *PendingCommit) Clear() error {
	if err := os.Remove(pc.headPath); err != nil {
		return err
	}
	return os.Remove(pc.messagePath)
}
