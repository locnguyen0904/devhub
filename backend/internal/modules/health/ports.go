package health

import "context"

// Pinger reports whether a dependency answers.
//
// Declared here, in the consuming module, rather than exported by the packages
// that implement it — see docs/01-architecture.md §4. It has two implementations
// from the start: the Postgres repository below and the Redis client.
type Pinger interface {
	Ping(ctx context.Context) error
}
