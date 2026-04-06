package fs

import "errors"

var (
	// ErrIsDirectory is returned when a file operation is attempted on a directory.
	ErrIsDirectory = errors.New("is a directory")

	// ErrNotSupported is returned for unsupported operations.
	ErrNotSupported = errors.New("operation not supported")

	// ErrNotFound is returned when a path does not exist.
	ErrNotFound = errors.New("not found")
)
