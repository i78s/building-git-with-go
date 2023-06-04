package database

type LockDeniedError struct {
	Message string
}

func (e *LockDeniedError) Error() string {
	return e.Message
}

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
