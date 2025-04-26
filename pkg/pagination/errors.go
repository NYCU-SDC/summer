package pagination

import "errors"

var (
	ErrInvalidPageOrSize   = errors.New("invalid page number or size")
	ErrInvalidSortingField = errors.New("invalid sorting field")
)
