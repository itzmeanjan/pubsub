package pubsub

import (
	"io"
	"sync"
)

type publishRequest struct {
	Message      *Message
	ResponseChan chan uint64
}

type consumptionRequest struct {
	Id           uint64
	ResponseChan chan bool
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
	lock   *sync.RWMutex
	buffer []*PublishedMessage
	Writer io.Writer
}
