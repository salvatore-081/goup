package models

import "fmt"

type ConflictError struct {
	HTTPCode int
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf("%d", e.HTTPCode)
}
