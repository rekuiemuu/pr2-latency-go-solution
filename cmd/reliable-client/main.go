package main

import (
	"encoding/csv"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strconv"
	"time"

	"go-algo/protocol"
	"go-algo/reliability"
	"go-algo/transport"
)

func main() {
	address := flag.String("addr", "127.0.0.1:19200", "server address")
	experiment := flag.String("experiment", "baseline", "experiment name")
	count := flag.Int("count", 50, "number of reliable commands")
	maxAttempts := flag.Uint("max-attempts", 4, "maximum send attempts")
	output := flag.String("out", "docs/reliability_samples.csv", "CSV path")
	appendCSV := flag.Bool("append", false, "append to CSV")
	flag.Parse()

	udp, err := transport.Dial(*address)
	if err != nil {
		log.Fatal(err)
	}
	defer udp.Close()

	timeout := &reliability.AdaptiveTimeout{}
	warmup(udp, timeout)
	channel := reliability.NewChannel(uint32(*maxAttempts))
	samples := make([]reliability.Sample, 0, *count)

	for i := 1; i <= *count; i++ {
		sequence := uint16(i)
		packet := protocol.Packet{Type: protocol.Shoot, Sequence: sequence, RequiresAck: true, Payload: []byte("shot")}
		raw, err := protocol.Encode(packet)
		if err != nil {
			log.Fatal(err)
		}
		now := time.Now()
		if err := channel.OnSent(sequence, raw, uint64(now.UnixMicro())); err != nil {
			log.Fatal(err)
		}
		if err := udp.Send(raw); err != nil {
			log.Fatal(err)
		}

		for channel.PendingCount() > 0 {
			replyRaw, receiveErr := udp.Receive(10 * time.Millisecond)
			if receiveErr == nil {
				reply, decodeErr := protocol.Decode(replyRaw)
				if decodeErr == nil && reply.Type == protocol.Ack {
					result := channel.OnAckReceived(reply.AcknowledgedSequence, uint64(time.Now().UnixMicro()))
					if result.Found {
						ackMs := float64(result.ElapsedUs) / 1000
						if result.FirstAttempt { // Karn: retransmitted packets do not update RTT/RTO.
							timeout.OnSample(ackMs)
						}
						samples = append(samples, reliability.Sample{Experiment: *experiment, Sequence: sequence, Attempts: result.Attempts, AckMs: ackMs, RtoMs: timeout.RtoMs()})
						fmt.Printf("ACK seq=%d attempts=%d time=%.2fms RTO=%.2fms\n", sequence, result.Attempts, ackMs, timeout.RtoMs())
					}
				}
			} else if !isTimeout(receiveErr) {
				log.Fatal(receiveErr)
			}

			beforeFailed := channel.FailedCount()
			for _, pending := range channel.CollectForRetransmission(uint64(time.Now().UnixMicro()), uint64(timeout.RtoMs()*1000)) {
				if err := udp.Send(pending.RawBytes); err != nil {
					log.Fatal(err)
				}
				fmt.Printf("RETRY seq=%d attempt=%d\n", pending.Sequence, pending.Attempts)
			}
			if channel.FailedCount() > beforeFailed {
				failed := channel.Failed()[channel.FailedCount()-1]
				samples = append(samples, reliability.Sample{Experiment: *experiment, Sequence: failed.Sequence, Attempts: failed.Attempts, Failed: true, RtoMs: timeout.RtoMs()})
				fmt.Printf("FAILED seq=%d after %d attempts\n", failed.Sequence, failed.Attempts)
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := writeSamples(*output, samples, *appendCSV); err != nil {
		log.Fatal(err)
	}
	stats := reliability.Calculate(samples)
	fmt.Printf("RESULT avg_attempts=%.3f first=%.1f%% failed=%.1f%% avg_ack=%.2fms final_RTO=%.2fms\n", stats.AverageAttempts, stats.FirstAttemptPct, stats.FailedPct, stats.AverageAckMs, stats.FinalRtoMs)
}

func warmup(udp *transport.UDP, timeout *reliability.AdaptiveTimeout) {
	for i := 0; i < 5; i++ {
		sequence := uint16(60000 + i)
		started := time.Now()
		raw, _ := protocol.Encode(protocol.Packet{Type: protocol.Ping, Sequence: sequence, ClientSendUs: uint64(started.UnixMicro())})
		if udp.Send(raw) != nil {
			continue
		}
		response, err := udp.Receive(time.Second)
		if err != nil {
			continue
		}
		packet, err := protocol.Decode(response)
		if err == nil && packet.Type == protocol.Pong && packet.Sequence == sequence {
			timeout.OnSample(float64(time.Since(started).Microseconds()) / 1000)
		}
	}
}

func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func writeSamples(path string, samples []reliability.Sample, appendMode bool) error {
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}
	file, err := os.OpenFile(path, flags, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := csv.NewWriter(file)
	defer writer.Flush()
	if !appendMode {
		writer.Write([]string{"experiment", "sequence", "attempts", "first_attempt", "failed", "time_to_ack_ms", "rto_ms"})
	}
	for _, sample := range samples {
		writer.Write([]string{sample.Experiment, strconv.Itoa(int(sample.Sequence)), strconv.Itoa(int(sample.Attempts)), strconv.FormatBool(!sample.Failed && sample.Attempts == 1), strconv.FormatBool(sample.Failed), fmt.Sprintf("%.3f", sample.AckMs), fmt.Sprintf("%.3f", sample.RtoMs)})
	}
	return writer.Error()
}
