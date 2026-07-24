package health

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

type stubPinger struct{ err error }

func (s stubPinger) Ping(context.Context) error { return s.err }

func TestServiceCheck(t *testing.T) {
	down := errors.New("connection refused")

	tests := []struct {
		name      string
		postgres  error
		redis     error
		want      []Check
		wantReady bool
	}{
		{
			name: "reports ready when every dependency answers",
			want: []Check{
				{Name: DepPostgres, OK: true},
				{Name: DepRedis, OK: true},
			},
			wantReady: true,
		},
		{
			name:     "reports not ready when postgres is down",
			postgres: down,
			want: []Check{
				{Name: DepPostgres, OK: false, Reason: down.Error()},
				{Name: DepRedis, OK: true},
			},
			wantReady: false,
		},
		{
			name:     "keeps probing after a failure so every dependency is reported",
			postgres: down,
			redis:    down,
			want: []Check{
				{Name: DepPostgres, OK: false, Reason: down.Error()},
				{Name: DepRedis, OK: false, Reason: down.Error()},
			},
			wantReady: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newService(stubPinger{err: tt.postgres}, stubPinger{err: tt.redis})

			got := svc.Check(t.Context())

			if diff := cmp.Diff(tt.want, got.Checks, cmpopts.EquateEmpty()); diff != "" {
				t.Errorf("Check() checks mismatch (-want +got):\n%s", diff)
			}
			if got.Ready() != tt.wantReady {
				t.Errorf("Check().Ready() = %v, want %v", got.Ready(), tt.wantReady)
			}
		})
	}
}
