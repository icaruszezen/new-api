package modeliqcheck

import (
	"testing"

	"github.com/QuantumNous/new-api/model"

	"github.com/stretchr/testify/assert"
)

func TestIsDue(t *testing.T) {
	const now = int64(1_700_000_000)
	hourly := Check{Id: "c1", Enabled: true, IntervalMinutes: 60}

	cases := []struct {
		name  string
		check Check
		state model.ModelIQCheckState
		want  bool
	}{
		{
			name:  "disabled check never runs",
			check: Check{Id: "c1", Enabled: false, IntervalMinutes: 60},
			state: model.ModelIQCheckState{RunRequestedAt: now},
			want:  false,
		},
		{
			name:  "never started runs immediately",
			check: hourly,
			want:  true,
		},
		{
			name:  "within interval waits",
			check: hourly,
			state: model.ModelIQCheckState{LastStartedAt: now - 60*60 + 1, LastFinishedAt: now - 60*60 + 30},
			want:  false,
		},
		{
			name:  "interval measured from the last start",
			check: hourly,
			state: model.ModelIQCheckState{LastStartedAt: now - 60*60, LastFinishedAt: now - 10},
			want:  true,
		},
		{
			name:  "run request after the last start forces a run",
			check: hourly,
			state: model.ModelIQCheckState{LastStartedAt: now - 60, LastFinishedAt: now - 30, RunRequestedAt: now - 5},
			want:  true,
		},
		{
			name:  "run request already consumed by a later start",
			check: hourly,
			state: model.ModelIQCheckState{LastStartedAt: now - 60, RunRequestedAt: now - 120},
			want:  false,
		},
		{
			name:  "unset interval uses the default",
			check: Check{Id: "c1", Enabled: true},
			state: model.ModelIQCheckState{LastStartedAt: now - int64(DefaultIntervalMinutes)*60},
			want:  true,
		},
		{
			name:  "unset interval still waits for the default",
			check: Check{Id: "c1", Enabled: true},
			state: model.ModelIQCheckState{LastStartedAt: now - int64(DefaultIntervalMinutes)*60 + 1},
			want:  false,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.want, IsDue(testCase.check, testCase.state, now))
		})
	}
}
