package reliability

import (
	"errors"
	"sort"
)

type PendingPacket struct {
	Sequence      uint16
	RawBytes      []byte
	FirstSentAtUs uint64
	LastSentAtUs  uint64
	Attempts      uint32
}

type AckResult struct {
	Found        bool
	Attempts     uint32
	ElapsedUs    uint64
	FirstAttempt bool
}

type ReliableChannel struct {
	maxAttempts uint32
	pending     map[uint16]*PendingPacket
	failed      []PendingPacket
}

func NewChannel(maxAttempts uint32) *ReliableChannel {
	if maxAttempts == 0 {
		maxAttempts = 1
	}
	return &ReliableChannel{maxAttempts: maxAttempts, pending: make(map[uint16]*PendingPacket)}
}

func (c *ReliableChannel) OnSent(sequence uint16, raw []byte, nowUs uint64) error {
	if _, exists := c.pending[sequence]; exists {
		return errors.New("sequence already pending")
	}
	c.pending[sequence] = &PendingPacket{Sequence: sequence, RawBytes: append([]byte(nil), raw...), FirstSentAtUs: nowUs, LastSentAtUs: nowUs, Attempts: 1}
	return nil
}

func (c *ReliableChannel) OnAckReceived(sequence uint16, nowUs uint64) AckResult {
	p := c.pending[sequence]
	if p == nil {
		return AckResult{}
	}
	delete(c.pending, sequence)
	return AckResult{Found: true, Attempts: p.Attempts, ElapsedUs: nowUs - p.FirstSentAtUs, FirstAttempt: p.Attempts == 1}
}

func (c *ReliableChannel) CollectForRetransmission(nowUs, rtoUs uint64) []*PendingPacket {
	keys := make([]int, 0, len(c.pending))
	for seq := range c.pending {
		keys = append(keys, int(seq))
	}
	sort.Ints(keys)
	var due []*PendingPacket
	for _, key := range keys {
		seq := uint16(key)
		p := c.pending[seq]
		if nowUs-p.LastSentAtUs < rtoUs {
			continue
		}
		if p.Attempts >= c.maxAttempts {
			c.failed = append(c.failed, *p)
			delete(c.pending, seq)
			continue
		}
		p.Attempts++
		p.LastSentAtUs = nowUs
		due = append(due, p)
	}
	return due
}

func (c *ReliableChannel) PendingCount() int       { return len(c.pending) }
func (c *ReliableChannel) FailedCount() int        { return len(c.failed) }
func (c *ReliableChannel) Failed() []PendingPacket { return append([]PendingPacket(nil), c.failed...) }
