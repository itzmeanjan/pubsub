package pubsub

import (
	"io"
	"time"
)

// Message - Publisher showing intent of publishing arbitrary byte slice to topics
type Message struct {
	Topics []string
	Data   Binary
}

type PublishedMessage struct {
	Topic string
	Data  Binary
}

type publishRequest struct {
	Message      *Message
	BlockFor     time.Duration
	ResponseChan chan uint64
}

type subscriptionRequest struct {
	Id           uint64
	info         *subscriberInfo
	Topics       []string
	ResponseChan chan uint64
}

type unsubscriptionRequest struct {
	Id           uint64
	Topics       []string
	ResponseChan chan uint64
}

type subscriberInfo struct {
	Ping   chan struct{}
	Writer io.Writer
}
