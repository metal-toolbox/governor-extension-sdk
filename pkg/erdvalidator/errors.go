package erdvalidator

import (
	"errors"
	"fmt"
)

// ErrERDValidationFailed is returned when ERD validation fails
var ErrERDValidationFailed = errors.New("ERD validation failed")

func mkValidationErr(msg string) error {
	return fmt.Errorf("%w: %s", ErrERDValidationFailed, msg)
}
