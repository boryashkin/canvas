package pubsub

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

type ConnectionSubscriptionsMap struct {
	subs         sync.Map
	topics       sync.Map
	topicCounter uint16

	globalCounter *atomic.Uint32
	allChans      sync.Map
	commonChan    chan struct {
		string
		uint64
	}
}

func NewConnectionSubscriptionsMap() ConnSubs {
	commonBus := make(chan struct {
		string
		uint64
	})
	var globalCounter atomic.Uint32
	globalCounter.Store(0)

	cs := &ConnectionSubscriptionsMap{}
	cs.subs = sync.Map{}
	cs.topics = sync.Map{}
	cs.topicCounter = 0
	cs.globalCounter = &globalCounter
	cs.allChans = sync.Map{}
	cs.commonChan = commonBus

	return cs
}
func (cs *ConnectionSubscriptionsMap) Subscribe(connectionID uint32, topic string) {
	cs.subs.Store(connectionID, topic)
}
func (cs *ConnectionSubscriptionsMap) Unsubscribe(connectionID uint32, topic string) {
	cs.unsubscribeAll(connectionID)
}

func (cs *ConnectionSubscriptionsMap) DeleteConnection(connectionID uint32) {
	cs.unsubscribeAll(connectionID)
	ch, ok := cs.allChans.Load(connectionID)
	if ok {
		cs.allChans.Delete(connectionID)
		close(ch.(chan uint64))
	}
}

// a chan has only values for now, as we have only one topic per connection
// todo: make connection space renewable or not bounded
func (cs *ConnectionSubscriptionsMap) GetNewConnectionIdAndChan() (uint32, chan uint64, error) {
	cid := cs.globalCounter.Add(1) - 1
	if cid >= maxUint16 {
		return 0, nil, errors.New("max connections exceede")
	}

	return cid, cs.getConnectionChan(cid), nil
}

func (cs *ConnectionSubscriptionsMap) Publish(topic string, value uint64) {
	cs.commonChan <- struct {
		string
		uint64
	}{topic, value}
}

// run in a goroutine
func (cs *ConnectionSubscriptionsMap) RunNotificationLoop(ctx context.Context) {
	for {
		select {
		case v := <-cs.commonChan:
			for i := uint32(0); i <= cs.globalCounter.Load(); i++ {
				if cs.getTopic(i) != v.string {
					continue
				}
				ch, ok := cs.allChans.Load(i)
				if ok && ch != nil {
					ch.(chan uint64) <- v.uint64
				}
			}
		case <-ctx.Done():
			close(cs.commonChan)
			return
		}
	}
}

// don't use directly without locks
func (cs *ConnectionSubscriptionsMap) unsubscribeAll(connectionID uint32) {
	cs.subs.Delete(connectionID)
}
func (cs *ConnectionSubscriptionsMap) getTopic(connectionID uint32) string {
	val, ok := cs.subs.Load(connectionID)
	if !ok {
		return ""
	}

	return val.(string)
}
func (cs *ConnectionSubscriptionsMap) getConnectionChan(connectionID uint32) chan uint64 {
	ch, ok := cs.allChans.Load(connectionID)
	if !ok {
		ch = make(chan uint64)
		cs.allChans.Store(connectionID, ch)
	}

	return ch.(chan uint64)
}
