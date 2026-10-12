package reliability

import "testing"

func TestCalculate(t *testing.T) {
	s := Calculate([]Sample{{Attempts: 1, AckMs: 10, RtoMs: 100}, {Attempts: 2, AckMs: 30, RtoMs: 120}, {Attempts: 3, Failed: true, RtoMs: 150}})
	if s.AverageAttempts != 2 || s.FirstAttemptPct < 33.3 || s.FirstAttemptPct > 33.4 || s.FailedPct < 33.3 || s.FailedPct > 33.4 || s.AverageAckMs != 20 || s.FinalRtoMs != 150 {
		t.Fatalf("unexpected statistics: %+v", s)
	}
}
