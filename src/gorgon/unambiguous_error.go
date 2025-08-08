package gorgon

func WrapUnambiguousError(err error) error {
	return &unambiguousError{err}
}

func IsUnambiguousError(err error) bool {
	_, ok := err.(*unambiguousError)
	return ok
}

type unambiguousError struct {
	wrapped error
}

func (e *unambiguousError) Error() string {
	return e.wrapped.Error()
}

func (e *unambiguousError) Unwrap() error {
	return e.wrapped
}
