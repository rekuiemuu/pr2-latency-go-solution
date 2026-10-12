package protocol

import "testing"

func TestRoundTrip(t *testing.T) {
	for _, want := range []Packet{
		{Type: Ping, Sequence: 42, ClientSendUs: 12345},
		{Type: Pong, Sequence: 42, ClientSendUs: 12345, ServerReceiveUs: 200, ServerSendUs: 300},
		{Type: Shoot, Sequence: 7, RequiresAck: true, Payload: []byte("shot")},
		{Type: Ack, Sequence: 8, AcknowledgedSequence: 7},
	} {
		b, err := Encode(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Decode(b)
		if err != nil {
			t.Fatal(err)
		}
		if got.Type != want.Type || got.Sequence != want.Sequence || got.ClientSendUs != want.ClientSendUs || got.ServerReceiveUs != want.ServerReceiveUs || got.ServerSendUs != want.ServerSendUs || got.RequiresAck != want.RequiresAck || got.AcknowledgedSequence != want.AcknowledgedSequence || string(got.Payload) != string(want.Payload) {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	}
}

func TestBadPackets(t *testing.T) {
	good, _ := Encode(Packet{Type: Ping})
	cases := [][]byte{good[:3], good[:len(good)-1]}
	for _, change := range []struct {
		index int
		value byte
	}{{0, 99}, {5, 99}, {4, 9}} {
		b := append([]byte(nil), good...)
		b[change.index] = change.value
		cases = append(cases, b)
	}
	for _, b := range cases {
		if _, err := Decode(b); err == nil {
			t.Fatalf("accepted %v", b)
		}
	}
	badAck, _ := Encode(Packet{Type: Ack, AcknowledgedSequence: 1})
	badAck[7] = 2
	if _, err := Decode(badAck); err == nil {
		t.Fatal("invalid requiresAck accepted")
	}
}
