package pubsub

import (
	"context"
	"io"
	"sync"
)

// Subscriber - Uniquely identifiable subscriber with multiple
// subscribed topics from where it wishes to listen from over single channel
type Subscriber struct {
	id     uint64
	reader io.Reader
	writer io.Writer
	ping   chan struct{}
	mLock  *sync.RWMutex
	tLock  *sync.RWMutex
	topics map[string]bool
	buffer []*PublishedMessage
	hub    *PubSub
}

func (s *Subscriber) start(ctx context.Context, started chan struct{}) {
	close(started)

	for {

		select {
		case <-ctx.Done():
			return

		case <-s.ping:
			t := new(String)
			if _, err := t.ReadFrom(s.reader); err != nil {
				continue
			}

			b := new(Binary)
			if _, err := b.ReadFrom(s.reader); err != nil {
				continue
			}

			s.mLock.Lock()
			s.buffer = append(s.buffer, &PublishedMessage{Topic: t.String(), Data: *b})
			s.mLock.Unlock()

		}

	}
}

func (s *Subscriber) Next() *PublishedMessage {
	s.mLock.Lock()
	defer s.mLock.Unlock()

	if len(s.buffer) == 0 {
		return nil
	}

	msg := s.buffer[0]
	n := len(s.buffer)

	copy(s.buffer[:], s.buffer[1:])
	s.buffer[n-1] = nil
	s.buffer = s.buffer[:n-1]

	return msg
}

func (s *Subscriber) AddSubscription(topics ...string) (bool, uint64) {
	if len(topics) == 0 {
		return s.hub.Alive, 0
	}

	s.tLock.Lock()
	defer s.tLock.Unlock()

	for i := 0; i < len(topics); i++ {
		s.topics[topics[i]] = true
	}

	return s.hub.addSubscription(&SubscriptionRequest{
		Id:     s.id,
		Ping:   s.ping,
		Writer: s.writer,
		Topics: topics,
	})
}

func (s *Subscriber) Unsubscribe(topics ...string) (bool, uint64) {
	if len(topics) == 0 {
		return s.hub.Alive, 0
	}

	s.tLock.Lock()
	defer s.tLock.Unlock()

	for i := 0; i < len(topics); i++ {
		s.topics[topics[i]] = false
	}

	return s.hub.unsubscribe(&UnsubscriptionRequest{
		Id:     s.id,
		Topics: topics,
	})
}

func (s *Subscriber) UnsubscribeAll() (bool, uint64) {
	topics := make([]string, 0, len(s.topics))

	s.tLock.RLock()
	for k, v := range s.topics {
		if v {
			topics = append(topics, k)
		}
	}
	s.tLock.RUnlock()

	return s.Unsubscribe(topics...)
}
