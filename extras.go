package pubsub

import "time"

// Message - Publisher showing intent of publishing arbitrary byte slice to topics
type Message struct {
	Topics []string
	Data   []byte
}

// PublishRequest - Publisher will show interest of publication using this form,
// while receiving how many subscribers it published to
type PublishRequest struct {
	Message      *Message
	BlockFor     time.Duration
	ResponseChan chan uint64
}

// PublishedMessage - Once a message is published on a topic, subscriber to receive it in this form
type PublishedMessage struct {
	Topic string
	Data  []byte
}

// SubscriptionRequest - Subscriber to send topic subscription request in this form,
// will also receive how many topics were successfully subscribed to
type SubscriptionRequest struct {
	Subscriber   *Subscriber
	ResponseChan chan uint64
}

// UnsubscriptionRequest - Topic unsubscription request to be sent in this form
// will also receive how many of them were successfully unsubscribed from
type UnsubscriptionRequest struct {
	Id           uint64
	Topics       []string
	ResponseChan chan uint64
}
