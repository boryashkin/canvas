package pubsub

import "testing"

func TestSubscribe(t *testing.T) {
	cs := NewConnectionSubscriptions()

	cs.Subscribe(1, "topic")
}
func TestGetNewConnectionIdAndChan(t *testing.T) {
	cs := NewConnectionSubscriptions()

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

func TestGetNewConnectionIdAndChanMax(t *testing.T) {
	cs := NewConnectionSubscriptions()

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

// -benchtime=65534x; results: 65534 380-450 ns/op
func BenchmarkSubUnsubMax(t *testing.B) {
	cs := NewConnectionSubscriptions()

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

// -benchtime=65534x; results: 65534 222-233 ns/op
func BenchmarkSubDeleteMax(t *testing.B) {
	cs := NewConnectionSubscriptions()

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
