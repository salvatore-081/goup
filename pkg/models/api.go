package models

import "fmt"

type Header struct {
	XAPIKey string `header:"X-API-Key"`
}

type Default struct {
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

type HTTPError struct {
	Status int
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("%d", e.Status)
}

func (e HTTPError) Equal(i error) bool {
	return fmt.Sprintf("%d", e.Status) == i.Error()
}
