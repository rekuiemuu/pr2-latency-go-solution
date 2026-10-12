package protocol

import (
	"encoding/binary"
	"errors"
)

const Version uint16 = 1
const HeaderSize = 8
const (
	Ping     byte = 1
	Pong     byte = 2
	Movement byte = 3
	Shoot    byte = 4
	Ack      byte = 5
)

type Packet struct {
	Type                 byte
	Sequence             uint16
	ClientSendUs         uint64
	ServerReceiveUs      uint64
	ServerSendUs         uint64
	RequiresAck          bool
	Payload              []byte
	AcknowledgedSequence uint16
}

func Encode(p Packet) ([]byte, error) {
	size := 0
	switch p.Type {
	case Ping:
		size = 8
	case Pong:
		size = 24
	case Movement, Shoot:
		size = len(p.Payload)
	case Ack:
		size = 2
	default:
		return nil, errors.New("unsupported packet type")
	}
	b := make([]byte, HeaderSize+size)
	b[0] = p.Type
	binary.BigEndian.PutUint16(b[1:3], p.Sequence)
	binary.BigEndian.PutUint16(b[3:5], uint16(size))
	binary.BigEndian.PutUint16(b[5:7], Version)
	if p.RequiresAck {
		b[7] = 1
	}
	payload := b[HeaderSize:]
	switch p.Type {
	case Ping, Pong:
		binary.BigEndian.PutUint64(payload[0:8], p.ClientSendUs)
	case Movement, Shoot:
		copy(payload, p.Payload)
	case Ack:
		binary.BigEndian.PutUint16(payload, p.AcknowledgedSequence)
	}
	if p.Type == Pong {
		binary.BigEndian.PutUint64(payload[8:16], p.ServerReceiveUs)
		binary.BigEndian.PutUint64(payload[16:24], p.ServerSendUs)
	}
	return b, nil
}

func Decode(b []byte) (Packet, error) {
	if len(b) < HeaderSize {
		return Packet{}, errors.New("short header")
	}
	kind := b[0]
	if kind < Ping || kind > Ack {
		return Packet{}, errors.New("invalid packet type")
	}
	if binary.BigEndian.Uint16(b[5:7]) != Version {
		return Packet{}, errors.New("invalid version")
	}
	size := int(binary.BigEndian.Uint16(b[3:5]))
	if size != len(b)-HeaderSize {
		return Packet{}, errors.New("payload size mismatch")
	}
	if b[7] > 1 {
		return Packet{}, errors.New("invalid requiresAck")
	}
	if kind == Ping && size != 8 || kind == Pong && size != 24 || kind == Ack && size != 2 {
		return Packet{}, errors.New("invalid payload length")
	}
	p := Packet{Type: kind, Sequence: binary.BigEndian.Uint16(b[1:3]), RequiresAck: b[7] == 1}
	payload := b[HeaderSize:]
	switch kind {
	case Ping, Pong:
		p.ClientSendUs = binary.BigEndian.Uint64(payload[0:8])
	case Movement, Shoot:
		p.Payload = append([]byte(nil), payload...)
	case Ack:
		p.AcknowledgedSequence = binary.BigEndian.Uint16(payload)
	}
	if kind == Pong {
		p.ServerReceiveUs = binary.BigEndian.Uint64(payload[8:16])
		p.ServerSendUs = binary.BigEndian.Uint64(payload[16:24])
	}
	return p, nil
}
