package npmregistry

import "errors"

// ErrFileNotFound is returned by FetchTarballFile when the requested path
// does not exist inside the tarball.
var ErrFileNotFound = errors.New("npmregistry: file not found in tarball")
