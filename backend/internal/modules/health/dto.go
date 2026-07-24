package health

// LiveOutput is the liveness response. It carries no dependency information on
// purpose: liveness answers "is the process running", and a probe that fails
// because Postgres is down would get the container restarted for no reason.
type LiveOutput struct {
	Body struct {
		Status string `json:"status" doc:"Always \"ok\" while the process is alive" example:"ok"`
	}
}

// ReadyOutput is the readiness response.
type ReadyOutput struct {
	Body struct {
		Status string `json:"status" doc:"\"ok\" when every dependency answers" example:"ok"`
		// nullable:"false" because toCheckDTOs always allocates. Without it huma
		// marks the slice nullable — a nil Go slice does serialise to null — and
		// the frontend inherits a null check that can never fire.
		Checks []CheckDTO `json:"checks" nullable:"false" doc:"Per-dependency probe results"`
	}
}

// CheckDTO is one dependency probe as sent to clients.
type CheckDTO struct {
	Name string `json:"name" doc:"Dependency name" example:"postgres"`
	OK   bool   `json:"ok"   doc:"Whether the dependency answered"`
}

func toCheckDTOs(checks []Check) []CheckDTO {
	out := make([]CheckDTO, 0, len(checks))
	for _, c := range checks {
		out = append(out, CheckDTO{Name: c.Name, OK: c.OK})
	}
	return out
}
