package main

import (
	"flag"
	"log"
	"math/rand"
	"net"
	"time"

	"go-algo/protocol"
	"go-algo/reliability"
	"go-algo/transport"
)

func main() {
	address := flag.String("addr", "127.0.0.1:19200", "listen address")
	mode := flag.String("mode", "baseline", "baseline, loss_5, loss_10, loss_20, delay_100_loss_5, jitter_loss_10")
	seed := flag.Int64("seed", 42, "random seed")
	flag.Parse()

	server, err := transport.Listen(*address)
	if err != nil {
		log.Fatal(err)
	}
	defer server.Close()

	rng := rand.New(rand.NewSource(*seed))
	processed := reliability.NewRecentWindow(1024)
	arrivals := make(map[uint16]int)
	log.Printf("reliable UDP server: %s, mode=%s", *address, *mode)

	for {
		raw, client, err := server.Receive()
		if err != nil {
			log.Printf("receive: %v", err)
			continue
		}
		packet, err := protocol.Decode(raw)
		if err != nil {
			log.Printf("invalid packet from %s: %v", client, err)
			continue
		}

		if packet.Type == protocol.Ping {
			now := uint64(time.Now().UnixMicro())
			reply := protocol.Packet{Type: protocol.Pong, Sequence: packet.Sequence, ClientSendUs: packet.ClientSendUs, ServerReceiveUs: now, ServerSendUs: now}
			sendLater(server, client, reply, responseDelay(*mode, rng))
			continue
		}
		if packet.Type != protocol.Shoot && packet.Type != protocol.Movement {
			continue
		}

		arrivals[packet.Sequence]++
		if shouldDrop(*mode, packet.Sequence, arrivals[packet.Sequence]) {
			log.Printf("DROP seq=%d", packet.Sequence)
			continue
		}
		if packet.RequiresAck {
			ack := protocol.Packet{Type: protocol.Ack, Sequence: packet.Sequence, AcknowledgedSequence: packet.Sequence}
			sendLater(server, client, ack, responseDelay(*mode, rng))
		}
		if processed.First(packet.Sequence) {
			log.Printf("PROCESS type=%d seq=%d payload=%q", packet.Type, packet.Sequence, packet.Payload)
		} else {
			log.Printf("DUPLICATE seq=%d (ACK repeated, effect skipped)", packet.Sequence)
		}
	}
}

func sendLater(server *transport.Server, client net.Addr, packet protocol.Packet, delay time.Duration) {
	raw, err := protocol.Encode(packet)
	if err != nil {
		log.Printf("encode response: %v", err)
		return
	}
	go func() {
		time.Sleep(delay)
		if err := server.Send(raw, client); err != nil {
			log.Printf("send response: %v", err)
		}
	}()
}

func shouldDrop(mode string, sequence uint16, arrival int) bool {
	if arrival != 1 {
		return false
	}
	switch mode {
	case "loss_5", "delay_100_loss_5":
		return sequence%20 == 0
	case "loss_10", "jitter_loss_10":
		return sequence%10 == 0
	case "loss_20":
		return sequence%5 == 0
	default:
		return false
	}
}

func responseDelay(mode string, rng *rand.Rand) time.Duration {
	switch mode {
	case "delay_100_loss_5":
		return 100 * time.Millisecond
	case "jitter_loss_10":
		return time.Duration(50+rng.Intn(101)) * time.Millisecond
	default:
		return 0
	}
}

func init() { log.SetFlags(log.Ltime | log.Lmicroseconds) }
