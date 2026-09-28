package pipeline

import "errors"

// ErrSchemaNotPublished is returned when a given Renovate release did not
// publish renovate-schema.json (true for releases older than roughly v34).
var ErrSchemaNotPublished = errors.New("pipeline: renovate-schema.json not published for this version")
