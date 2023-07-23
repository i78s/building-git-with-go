package lockfile

import (
	"building-git/lib/errors"
	"os"
)

type Lockfile struct {
	filePath string
	lockPath string
	Lock     *os.File
}

func NewLockfile(filePath string) *Lockfile {
	return &Lockfile{
		filePath: filePath,
		lockPath: filePath + ".lock",
	}
}

func (lf *Lockfile) HoldForUpdate() error {
	if lf.Lock == nil {
		lock, err := os.OpenFile(lf.lockPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			if os.IsExist(err) {
				return &errors.LockDeniedError{Message: err.Error()}
			}
			if os.IsNotExist(err) {
				return &errors.MissingParentError{Message: err.Error()}
			}
			if os.IsPermission(err) {
				return &errors.NoPermissionError{Message: err.Error()}
			}
			return err
		}
		lf.Lock = lock
	}
	return nil
}

func (lf *Lockfile) Write(data []byte) error {
	if err := lf.raiseOnStaleLock(); err != nil {
		return err
	}
	_, err := lf.Lock.Write(data)
	return err
}

func (lf *Lockfile) Commit() error {
	if err := lf.raiseOnStaleLock(); err != nil {
		return err
	}
	lf.Lock.Close()
	err := os.Rename(lf.lockPath, lf.filePath)
	if err == nil {
		lf.Lock = nil
	}
	return err
}

func (lf *Lockfile) Rollback() error {
	if err := lf.raiseOnStaleLock(); err != nil {
		return err
	}
	if err := lf.Lock.Close(); err != nil {
		return err
	}
	if err := os.Remove(lf.lockPath); err != nil {
		return err
	}
	lf.Lock = nil

	return nil
}

func (lf *Lockfile) raiseOnStaleLock() error {
	if lf.Lock == nil {
		return &errors.StaleLockError{Message: "Not holding lock on file: " + lf.lockPath}
	}
	return nil
}
