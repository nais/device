package humanerror

import "fmt"

type HumanError struct {
	original error
	Message  string
}

func (e *HumanError) Error() string {
	return fmt.Sprintf("%v: %v", e.Message, e.original)
}

func (e *HumanError) Unwrap() error {
	return e.original
}

func Errorf(format string, a ...any) error {
	return &HumanError{
		Message: fmt.Sprintf(format, a...),
	}
}

func Wrap(err error, format string, a ...any) error {
	if err == nil {
		return nil
	}
	return &HumanError{
		original: err,
		Message:  fmt.Sprintf(format, a...),
	}
}
