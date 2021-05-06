package pubsub

import (
	"io"
)

// Subscriber - Uniquely identifiable subscriber with multiple
// subscribed topics from where it wishes to listen from over single channel
type Subscriber struct {
	Id     uint64
	Reader io.Reader
	Writer io.Writer
	Topics map[string]bool
}

// Next - ...
func (s *Subscriber) Next() *PublishedMessage {
	b := new(Binary)
	if _, err := b.ReadFrom(s.Reader); err != nil {
		return nil
	}

	return &PublishedMessage{Data: *b}
}

// AddSubscription - Subscribe to topics using existing pub/sub client
func (s *Subscriber) AddSubscription(pubsub *PubSub, topics ...string) (bool, uint64) {
	return pubsub.AddSubscription(s, topics...)
}

// Unsubscribe - Unsubscribe from topics, if subscribed to them using this client
func (s *Subscriber) Unsubscribe(pubsub *PubSub, topics ...string) (bool, uint64) {
	return pubsub.Unsubscribe(s, topics...)
}

// UnsubcribeAll - Unsubscribes from all topics this client is currently subscribed to
func (s *Subscriber) UnsubscribeAll(pubsub *PubSub) (bool, uint64) {
	return pubsub.UnsubscribeAll(s)
}

// Close - Destroys subscriber
func (s *Subscriber) Close() bool {

	for {
		if msg := s.Next(); msg == nil {
			break
		}
	}

	for topic := range s.Topics {
		delete(s.Topics, topic)
	}

	return true
}
