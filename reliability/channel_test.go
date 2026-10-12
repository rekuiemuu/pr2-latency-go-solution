package reliability

import "testing"

func TestRetransmissionAfterRTO(t *testing.T) {
	c := NewChannel(3)
	if err := c.OnSent(1, []byte{4, 0, 1}, 0); err != nil {
		t.Fatal(err)
	}
	if got := c.CollectForRetransmission(50_000, 200_000); len(got) != 0 {
		t.Fatal(got)
	}
	got := c.CollectForRetransmission(250_000, 200_000)
	if len(got) != 1 || got[0].Attempts != 2 {
		t.Fatal(got)
	}
	if got := c.CollectForRetransmission(300_000, 200_000); len(got) != 0 {
		t.Fatal("returned too early")
	}
}

func TestAttemptsExhausted(t *testing.T) {
	c := NewChannel(2)
	c.OnSent(7, nil, 0)
	c.CollectForRetransmission(300_000, 200_000)
	c.CollectForRetransmission(600_000, 200_000)
	if c.FailedCount() != 1 || c.PendingCount() != 0 {
		t.Fatalf("pending=%d failed=%d", c.PendingCount(), c.FailedCount())
	}
}

func TestDuplicateAckIgnored(t *testing.T) {
	c := NewChannel(5)
	c.OnSent(3, nil, 100)
	if !c.OnAckReceived(3, 200).Found {
		t.Fatal("first ACK ignored")
	}
	if c.OnAckReceived(3, 300).Found {
		t.Fatal("duplicate ACK accepted")
	}
}

func TestAdaptiveTimeout(t *testing.T) {
	var a AdaptiveTimeout
	if a.RtoMs() != 1000 {
		t.Fatal(a.RtoMs())
	}
	a.OnSample(100)
	if a.RtoMs() != 300 {
		t.Fatal(a.RtoMs())
	}
	a.OnSample(200)
	if a.SrttMs() != 112.5 || a.RttVarMs() != 62.5 || a.RtoMs() != 362.5 {
		t.Fatalf("srtt=%v var=%v rto=%v", a.SrttMs(), a.RttVarMs(), a.RtoMs())
	}
}
