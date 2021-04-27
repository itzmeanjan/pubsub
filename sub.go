package pubsub

import "time"

type Subscriber struct {
	Id      uint64
	Channel chan *PublishedMessage
	Topics  map[string]bool
}

func (s *Subscriber) Next() *PublishedMessage {
	if len(s.Channel) != 0 {
		return <-s.Channel
	}

	return nil
}

func (s *Subscriber) BNext(delay time.Duration) *PublishedMessage {
	if data := s.Next(); data != nil {
		return data
	}

	<-time.After(delay)
	return s.Next()
}

func (s *Subscriber) AddSubscription(pubsub *PubSub, topics ...string) (bool, uint64) {
	return pubsub.AddSubscription(s, topics...)
}

func (s *Subscriber) Unsubscribe(pubsub *PubSub, topics ...string) (bool, uint64) {
	return pubsub.Unsubscribe(s, topics...)
}

func (s *Subscriber) UnsubcribeAll(pubsub *PubSub) (bool, uint64) {
	return pubsub.UnsubscribeAll(s)
}
