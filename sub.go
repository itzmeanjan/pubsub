package pubsub

import (
	"sync"
)

// Subscriber - Uniquely identifiable subscriber with multiple
// subscribed topics from where it wishes to listen from over ping channel
type Subscriber struct {
	id     uint64
	info   *subscriberInfo
	tLock  *sync.RWMutex
	topics map[string]bool
	hub    *PubSub
}

// Listener - Get notified when new message is received
func (s *Subscriber) Listener() chan struct{} {
	return s.info.ping
}

// Consumable - Checks whether any consumable messages
// in buffer or not [ concurrent-safe ]
func (s *Subscriber) Consumable() bool {
	s.info.lock.RLock()
	defer s.info.lock.RUnlock()

	return len(s.info.buffer) != 0
}

// Next - Attempt to consume oldest message living in buffer,
// by popping it out, in concurrent-safe manner
//
// If nothing exists, it'll return nil
func (s *Subscriber) Next() *PublishedMessage {
	if !s.Consumable() {
		return nil
	}

	s.info.lock.Lock()
	defer s.info.lock.Unlock()

	msg := s.info.buffer[0]
	n := len(s.info.buffer)

	copy(s.info.buffer[:], s.info.buffer[1:])
	s.info.buffer[n-1] = nil
	s.info.buffer = s.info.buffer[:n-1]

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

func (s *Subscriber) Destroy() bool {
	ok, _ := s.UnsubscribeAll()
	if ok {
		return s.hub.destroy(&destroyRequest{
			Id: s.id,
		})
	}

	return false
}
