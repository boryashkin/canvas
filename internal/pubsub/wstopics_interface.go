package pubsub

import (
	"context"
)

type ConnSubs interface {
	Subscribe(connectionID uint32, topic string)
	Unsubscribe(connectionID uint32, topic string)
	DeleteConnection(connectionID uint32)
	GetNewConnectionIdAndChan() (uint32, chan uint64, error)
	Publish(topic string, value uint64)
	RunNotificationLoop(ctx context.Context)
}
