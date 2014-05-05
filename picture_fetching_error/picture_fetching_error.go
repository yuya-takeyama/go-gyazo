package picture_fetching_error

import "fmt"

const (
	PictureNotFoundError = 0
	InternalMysqlError   = 1
)

type PictureFetchingError struct {
	s    string
	code int
}

func (err *PictureFetchingError) Error() string {
	return err.s
}

func (err *PictureFetchingError) IsPictureNotFound() bool {
	return err.code == PictureNotFoundError
}

func (err *PictureFetchingError) IsInternalMysqlError() bool {
	return err.code == InternalMysqlError
}

func New(s string, code int, err error) *PictureFetchingError {
	return &PictureFetchingError{fmt.Sprintf("%s: %s", s, err.Error()), code}
}
