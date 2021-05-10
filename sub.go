package pubsub

import (
	"context"
	"io"
	"sync"
)

// Subscriber - Uniquely identifiable subscriber with multiple
// subscribed topics from where it wishes to listen from over ping channel
type Subscriber struct {
	id       uint64
	reader   io.Reader
	info     *subscriberInfo
	mLock    *sync.RWMutex
	tLock    *sync.RWMutex
	topics   map[string]bool
	Listener chan struct{}
	buffer   []*PublishedMessage
	hub      *PubSub
}

// Next - Attempt to consume oldest message living in buffer,
// by popping it out, in concurrent-safe manner
func (s *Subscriber) Next() *PublishedMessage {
	s.mLock.RLock()
	if len(s.buffer) == 0 {
		s.mLock.RUnlock()
		return nil
	}
	s.mLock.RUnlock()

	s.mLock.Lock()
	defer s.mLock.Unlock()

	msg := s.buffer[0]
	n := len(s.buffer)

	copy(s.buffer[:], s.buffer[1:])
	s.buffer[n-1] = nil
	s.buffer = s.buffer[:n-1]

	return msg
}

// AddSubscription - Add subscriptions to more topics on-the-fly
func (s *Subscriber) AddSubscription(topics ...string) (bool, uint64) {
	s.tLock.Lock()
	defer s.tLock.Unlock()

	if len(topics) == 0 {
		return s.hub.IsAlive(), 0
	}

	for i := 0; i < len(topics); i++ {
		s.topics[topics[i]] = true
	}

	return s.hub.addSubscription(&subscriptionRequest{
		Id:     s.id,
		info:   s.info,
		Topics: topics,
	})
}

// Unsubscribe - Unsubscribe from specified subscribed topics
func (s *Subscriber) Unsubscribe(topics ...string) (bool, uint64) {
	s.tLock.Lock()
	defer s.tLock.Unlock()

	if len(topics) == 0 {
		return s.hub.IsAlive(), 0
	}

	for i := 0; i < len(topics); i++ {
		s.topics[topics[i]] = false
	}

	return s.hub.unsubscribe(&unsubscriptionRequest{
		Id:     s.id,
		Topics: topics,
	})
}

// UnsubscribeAll - Unsubscribe from all active subscribed topics
func (s *Subscriber) UnsubscribeAll() (bool, uint64) {
	s.tLock.RLock()
	topics := make([]string, 0, len(s.topics))

	for k, v := range s.topics {
		if v {
			topics = append(topics, k)
			s.topics[k] = false
		}
	}
	s.tLock.RUnlock()

	return s.Unsubscribe(topics...)
}

// start - Underlying message consumer from readable stream, starts
// working when notified to do so
func (s *Subscriber) start(ctx context.Context, started chan struct{}) {
	close(started)

	for {
		select {
		case <-ctx.Done():
			return

		case <-s.info.Ping:
			msg := new(PublishedMessage)
			if _, err := msg.ReadFrom(s.reader); err != nil {
				continue
			}

			s.mLock.Lock()
			s.buffer = append(s.buffer, msg)
			s.mLock.Unlock()

			// notify
			s.Listener <- struct{}{}

		}
	}
}
