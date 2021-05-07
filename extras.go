package pubsub

import "io"

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

type subscriberInfo struct {
	Ping   chan struct{}
	Writer io.Writer
}
