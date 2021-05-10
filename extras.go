package pubsub

import (
	"sync"
)

// Message - Publisher showing intent of publishing arbitrary byte slice to topics
type Message struct {
	Topics []string
	Data   []byte
}

// PublishedMessage - Subscriber will receive message for consumption in this form
type PublishedMessage struct {
	Topic string
	Data  []byte
}

type publishRequest struct {
	message      *Message
	responseChan chan uint64
}

type subscriptionRequest struct {
	id     uint64
	topics []string
}

type unsubscriptionRequest struct {
	id     uint64
	topics []string
}

type subscriberInfo struct {
	ping   chan struct{}
	lock   *sync.RWMutex
	buffer []*PublishedMessage
}
