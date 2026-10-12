package reliability

type Sample struct {
	Experiment string
	Sequence   uint16
	Attempts   uint32
	Failed     bool
	AckMs      float64
	RtoMs      float64
}

type Statistics struct {
	AverageAttempts float64
	FirstAttemptPct float64
	FailedPct       float64
	AverageAckMs    float64
	FinalRtoMs      float64
}

func Calculate(samples []Sample) Statistics {
	if len(samples) == 0 {
		return Statistics{}
	}
	var attempts uint64
	var first, failed, ackCount int
	var ackSum float64
	for _, sample := range samples {
		attempts += uint64(sample.Attempts)
		if !sample.Failed && sample.Attempts == 1 {
			first++
		}
		if sample.Failed {
			failed++
		} else {
			ackSum += sample.AckMs
			ackCount++
		}
	}
	stats := Statistics{
		AverageAttempts: float64(attempts) / float64(len(samples)),
		FirstAttemptPct: 100 * float64(first) / float64(len(samples)),
		FailedPct:       100 * float64(failed) / float64(len(samples)),
		FinalRtoMs:      samples[len(samples)-1].RtoMs,
	}
	if ackCount > 0 {
		stats.AverageAckMs = ackSum / float64(ackCount)
	}
	return stats
}
