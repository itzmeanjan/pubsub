package pubsub

import (
	"io"
	"sync"
)

// Subscriber - Uniquely identifiable subscriber with multiple
// subscribed topics from where it wishes to listen from over single channel
type Subscriber struct {
	Id             uint64
	Reader         io.Reader
	Writer         io.Writer
	mLock          *sync.RWMutex
	tLock          *sync.RWMutex
	NewMessageChan chan *PublishedMessage
	NextChan       chan chan *PublishedMessage
	NewTopicChan   chan []string
	Topics         map[string]bool
	Buffer         []*PublishedMessage
	Hub            *PubSub
}

func (s *Subscriber) Start() {
	for {
		b := new(Binary)
		if _, err := b.ReadFrom(s.Reader); err != nil {
			continue
		}

		s.mLock.Lock()
		s.Buffer = append(s.Buffer, &PublishedMessage{Data: *b})
		s.mLock.Unlock()
	}
}

func (s *Subscriber) Next() *PublishedMessage {
	s.mLock.Lock()
	defer s.mLock.Unlock()

	if len(s.Buffer) == 0 {
		return nil
	}

	msg := s.Buffer[0]
	n := len(s.Buffer)

	copy(s.Buffer[:], s.Buffer[1:])
	s.Buffer[n-1] = nil
	s.Buffer = s.Buffer[:n-1]

	return msg
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
