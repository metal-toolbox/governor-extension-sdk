package erdscli

import "errors"

var (
	// ErrValidatorMissingArgs is returned when required arguments are missing
	ErrValidatorMissingArgs = errors.New("missing required arguments")
	// ErrFailedToReadFiles is returned when files cannot be read
	ErrFailedToReadFiles = errors.New("failed to read files")
	// ErrFailedCreateFile is returned when a file cannot be created
	ErrFailedCreateFile = errors.New("failed to create file")
)
