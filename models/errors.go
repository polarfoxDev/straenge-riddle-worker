package models

// enum for riddle error types
const (
	ErrWordFill   = "WordFillError"
	ErrWordLength = "WordLengthError"
	ErrAmbiguity  = "AmbiguityError"
)

type RiddleError struct {
	ErrType string
	Message string
}

func (e *RiddleError) Error() string {
	return e.ErrType + ": " + e.Message
}
