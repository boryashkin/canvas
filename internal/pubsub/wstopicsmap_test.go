package pubsub

import "testing"

func TestSubscribeMap(t *testing.T) {
	cs := NewConnectionSubscriptionsMap()

	cs.Subscribe(1, "topic")
}
func TestGetNewConnectionIdAndChanMap(t *testing.T) {
	cs := NewConnectionSubscriptionsMap()

	_, ch, err := cs.GetNewConnectionIdAndChan()
	if err != nil {
		t.Error(err)
		t.FailNow()
	}
	if ch == nil {
		t.Error("nil channel")
		t.FailNow()
	}
}

func TestGetNewConnectionIdAndChanMaxMap(t *testing.T) {
	cs := NewConnectionSubscriptionsMap()

	for i := 0; i < int(maxUint16); i++ {
		_, ch, err := cs.GetNewConnectionIdAndChan()
		if err != nil {
			t.Error(i, err)
			t.FailNow()
		}
		if ch == nil {
			t.Error(i, "nil channel")
			t.FailNow()
		}
	}
	_, _, err := cs.GetNewConnectionIdAndChan()
	if err == nil {
		t.Error("last", "no error after max is exceeded")
	}
}

// -benchtime=65534x; results: 65534 1490-1700 ns/op
func BenchmarkSubUnsubMaxMap(t *testing.B) {
	cs := NewConnectionSubscriptionsMap()

	topic := "topic"
	for i := 0; i < t.N; i++ {
		cid, ch, err := cs.GetNewConnectionIdAndChan()
		if err != nil {
			t.Error(i, err)
			t.FailNow()
		}
		if ch == nil {
			t.Error(i, "nil channel")
			t.FailNow()
		}
		cs.Subscribe(cid, topic)
		cs.Unsubscribe(cid, topic)
	}
}

// -benchtime=65534x; results: 65534 1484-1506 ns/op
func BenchmarkSubDeleteMaxMap(t *testing.B) {
	cs := NewConnectionSubscriptionsMap()

	topic := "topic"
	for i := 0; i < t.N; i++ {
		cid, ch, err := cs.GetNewConnectionIdAndChan()
		if err != nil {
			t.Error(i, err)
			t.FailNow()
		}
		if ch == nil {
			t.Error(i, "nil channel")
			t.FailNow()
		}
		cs.Subscribe(cid, topic)
		cs.DeleteConnection(cid)
	}
}
