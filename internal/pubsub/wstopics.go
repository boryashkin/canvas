package pubsub

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
)

const maxUint16 = uint32(^uint16(0))

type ConnectionSubscriptions struct {
	sync.Mutex                     // todo profile and compare with sync.Map
	subs         map[uint32]string // todo profile and compare with sync.Map
	topics       map[string]uint16
	topicCounter uint16

	globalCounter *atomic.Uint32
	allChans      map[uint32]chan uint64 // todo profile and compare with sync.Map
	commonChan    chan struct {
		string
		uint64
	}
}

func NewConnectionSubscriptions() ConnSubs {
	allCh := make(map[uint32]chan uint64, 32)
	commonBus := make(chan struct {
		string
		uint64
	})
	var globalCounter atomic.Uint32
	globalCounter.Store(0)

	cs := &ConnectionSubscriptions{}
	cs.subs = make(map[uint32]string, 32)
	cs.topics = make(map[string]uint16, 32)
	cs.topicCounter = 0
	cs.globalCounter = &globalCounter
	cs.allChans = allCh
	cs.commonChan = commonBus

	return cs
}
func (cs *ConnectionSubscriptions) Subscribe(connectionID uint32, topic string) {
	cs.Lock()
	defer cs.Unlock()
	cs.subs[connectionID] = topic
}
func (cs *ConnectionSubscriptions) Unsubscribe(connectionID uint32, topic string) {
	cs.Lock()
	defer cs.Unlock()
	cs.unsubscribeAll(connectionID)
}

func (cs *ConnectionSubscriptions) DeleteConnection(connectionID uint32) {
	cs.Lock()
	defer cs.Unlock()
	cs.unsubscribeAll(connectionID)
	ch, ok := cs.allChans[connectionID]
	if ok {
		delete(cs.allChans, connectionID)
		close(ch)
	}
}

// a chan has only values for now, as we have only one topic per connection
// todo: make connection space renewable or not bounded
func (cs *ConnectionSubscriptions) GetNewConnectionIdAndChan() (uint32, chan uint64, error) {
	cid := cs.globalCounter.Add(1) - 1
	if cid >= maxUint16 {
		return 0, nil, errors.New("max connections exceede")
	}

	return cid, cs.getConnectionChan(cid), nil
}

func (cs *ConnectionSubscriptions) Publish(topic string, value uint64) {
	cs.commonChan <- struct {
		string
		uint64
	}{topic, value}
}

// run in a goroutine
func (cs *ConnectionSubscriptions) RunNotificationLoop(ctx context.Context) {
	for {
		select {
		case v := <-cs.commonChan:
			for i := uint32(0); i <= cs.globalCounter.Load(); i++ {
				if cs.getTopic(i) != v.string {
					continue
				}
				ch, ok := cs.allChans[i]
				if ok && ch != nil {
					ch <- v.uint64
				}
			}
		case <-ctx.Done():
			close(cs.commonChan)
			return
		}
	}
}

// don't use directly without locks
func (cs *ConnectionSubscriptions) unsubscribeAll(connectionID uint32) {
	delete(cs.subs, connectionID)
}
func (cs *ConnectionSubscriptions) getTopic(connectionID uint32) string {
	cs.Lock()
	defer cs.Unlock()
	return cs.subs[connectionID]
}
func (cs *ConnectionSubscriptions) getConnectionChan(connectionID uint32) chan uint64 {
	cs.Lock()
	defer cs.Unlock()

	ch, ok := cs.allChans[connectionID]
	if !ok {
		ch = make(chan uint64)
		cs.allChans[connectionID] = ch
	}

	return ch
}
