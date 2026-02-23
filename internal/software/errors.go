package software

import "errors"

var (
	// ErrSoftwareNotFound is returned when software is not found in config
	ErrSoftwareNotFound = errors.New("software not found")
)
