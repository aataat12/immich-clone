package media

import (
	"testing"
)

func TestVideoPlanAndActualTimestamps(t *testing.T) {
	for _, duration := range []int64{1, 11, 125000, 4000000, 86400 * 1e6} {
		targets := SampleTargets(duration)
		if len(targets) < 1 || len(targets) > 12 || targets[0] != 0 {
			t.Fatal("invalid sample count")
		}
		for i, v := range targets {
			if v < 0 || v >= duration || (i > 0 && v <= targets[i-1]) {
				t.Fatal("invalid sample interval")
			}
		}
	}
	if SampleTargets(0) != nil || SampleTargets(86400*1e6+1) != nil {
		t.Fatal("invalid duration accepted")
	}
	b := []byte(`{"frames":[{"best_effort_timestamp_time":"5.000000"},{"best_effort_timestamp_time":"5.125000"},{"best_effort_timestamp_time":"5.875000"}]}`)
	if v, e := chooseSample(b, 800000, 4000000, 5000000); e != nil || v != 875000 {
		t.Fatal("actual PTS selection", v, e)
	}
	if v, e := chooseSample(b, 124999, 125000, 5000000); e != nil || v != 0 {
		t.Fatal("short clip fallback", v, e)
	}
	for _, s := range []string{`{"frames":[{"best_effort_timestamp_time":"NaN"}]}`, `{"frames":[]}`, `invalid`} {
		if _, e := chooseSample([]byte(s), 0, 1000000, 0); e == nil {
			t.Fatal("invalid timestamps accepted")
		}
	}
}
