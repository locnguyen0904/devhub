package health

import (
	"context"
	"time"
)

// checkTimeout bounds each dependency probe. A readiness endpoint that hangs is
// worse than one that reports failure, because orchestrators read the timeout
// as "still starting" rather than "broken".
const checkTimeout = 2 * time.Second

// Dependency names as reported to clients.
const (
	DepPostgres = "postgres"
	DepRedis    = "redis"
)

// Check is the outcome of probing one dependency.
type Check struct {
	Name   string
	OK     bool
	Reason string // empty when OK
}

// Status aggregates every dependency probe.
type Status struct {
	Checks []Check
}

// Ready reports whether every dependency answered.
func (s Status) Ready() bool {
	for _, c := range s.Checks {
		if !c.OK {
			return false
		}
	}
	return true
}

// Service reports whether the API can serve traffic.
type Service interface {
	Check(ctx context.Context) Status
}

type dependency struct {
	name   string
	pinger Pinger
}

type service struct {
	deps []dependency
}

func newService(postgres, redis Pinger) *service {
	return &service{deps: []dependency{
		{name: DepPostgres, pinger: postgres},
		{name: DepRedis, pinger: redis},
	}}
}

// Check probes dependencies in order. Sequential is fine at two dependencies;
// each is bounded by checkTimeout so the worst case stays predictable.
func (s *service) Check(ctx context.Context) Status {
	checks := make([]Check, 0, len(s.deps))
	for _, d := range s.deps {
		probeCtx, cancel := context.WithTimeout(ctx, checkTimeout)
		err := d.pinger.Ping(probeCtx)
		cancel()

		c := Check{Name: d.name, OK: err == nil}
		if err != nil {
			c.Reason = err.Error()
		}
		checks = append(checks, c)
	}
	return Status{Checks: checks}
}
