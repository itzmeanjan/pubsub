package pubsub

import (
	"sync"
)

type publishRequest struct {
	Message      *Message
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

type destroyRequest struct {
	Id           uint64
	RepsonseChan chan bool
}

type subscriberInfo struct {
	ping   chan struct{}
	lock   *sync.RWMutex
	buffer []*PublishedMessage
}
