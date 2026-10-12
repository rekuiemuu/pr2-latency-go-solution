package reliability

import "math"

type AdaptiveTimeout struct {
	initialized bool
	srttMs      float64
	rttVarMs    float64
}

func (a *AdaptiveTimeout) OnSample(rttMs float64) {
	if rttMs < 0 {
		return
	}
	if !a.initialized {
		a.srttMs, a.rttVarMs, a.initialized = rttMs, rttMs/2, true
		return
	}
	a.rttVarMs = .75*a.rttVarMs + .25*math.Abs(a.srttMs-rttMs)
	a.srttMs = .875*a.srttMs + .125*rttMs
}

func (a *AdaptiveTimeout) RtoMs() float64 {
	if !a.initialized {
		return 1000
	}
	rto := a.srttMs + 4*a.rttVarMs
	if rto < 100 {
		return 100
	}
	if rto > 3000 {
		return 3000
	}
	return rto
}

func (a *AdaptiveTimeout) SrttMs() float64   { return a.srttMs }
func (a *AdaptiveTimeout) RttVarMs() float64 { return a.rttVarMs }
