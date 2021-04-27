package pubsub

import "time"

// Subscriber - Uniquely identifiable subscriber with multiple
// subscribed topics from where it wishes to listen from over single channel
type Subscriber struct {
	Id      uint64
	Channel chan *PublishedMessage
	Topics  map[string]bool
}

// Next - Read from channel if anything is immediately available
// otherwise just return i.e. it's non-blocking op
func (s *Subscriber) Next() *PublishedMessage {
	if len(s.Channel) != 0 {
		return <-s.Channel
	}

	return nil
}

// BNext - Read from channel, if something is available immediately
// otherwise wait for specified delay & attempt to read again where
// this step is non-blocking
func (s *Subscriber) BNext(delay time.Duration) *PublishedMessage {
	if data := s.Next(); data != nil {
		return data
	}

	<-time.After(delay)
	return s.Next()
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
func (s *Subscriber) UnsubcribeAll(pubsub *PubSub) (bool, uint64) {
	return pubsub.UnsubscribeAll(s)
}
