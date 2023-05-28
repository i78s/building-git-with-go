package jit

import "os"

type MissingParentError struct {
	Message string
}

func (e *MissingParentError) Error() string {
	return e.Message
}

type NoPermissionError struct {
	Message string
}

func (e *NoPermissionError) Error() string {
	return e.Message
}

type StaleLockError struct {
	Message string
}

func (e *StaleLockError) Error() string {
	return e.Message
}

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

func (lf *Lockfile) HoldForUpdate() (bool, error) {
	if lf.Lock == nil {
		lock, err := os.OpenFile(lf.lockPath, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
		if err != nil {
			if os.IsExist(err) {
				return false, nil
			}
			if os.IsNotExist(err) {
				return false, &MissingParentError{err.Error()}
			}
			if os.IsPermission(err) {
				return false, &NoPermissionError{err.Error()}
			}
			return false, err
		}
		lf.Lock = lock
	}
	return true, nil
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
		return &StaleLockError{"Not holding lock on file: " + lf.lockPath}
	}
	return nil
}