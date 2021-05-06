package pubsub

import (
	"io"
	"sync"
)

// Subscriber - Uniquely identifiable subscriber with multiple
// subscribed topics from where it wishes to listen from over single channel
type Subscriber struct {
	Id     uint64
	Reader io.Reader
	Writer io.Writer
	mLock  *sync.RWMutex
	tLock  *sync.RWMutex
	Topics map[string]bool
	Buffer []*PublishedMessage
	hub    *PubSub
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

func (s *Subscriber) AddSubscription(topics ...string) (bool, uint64) {
	for i := 0; i < len(topics); i++ {
		s.Topics[topics[i]] = true
	}

	return s.hub.addSubscription(&SubscriptionRequest{
		Id:     s.Id,
		Writer: s.Writer,
		Topics: topics,
	})
}

func (s *Subscriber) Unsubscribe(topics ...string) (bool, uint64) {
	for i := 0; i < len(topics); i++ {
		s.Topics[topics[i]] = false
	}

	return s.hub.unsubscribe(&UnsubscriptionRequest{
		Id:     s.Id,
		Topics: topics,
	})
}

func (s *Subscriber) UnsubscribeAll() (bool, uint64) {
	topics := make([]string, 0, len(s.Topics))

	for k := range s.Topics {
		topics = append(topics, k)
	}

	return s.Unsubscribe(topics...)
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
