package pubsub

import (
	"sync"
)

// PubSub - Pub/Sub Server i.e. holds which clients are subscribed to what topics,
// manages publishing messages to correct topics, handles (un-)subscription requests
//
// In other words state manager of Pub/Sub Broker
type PubSub struct {
	shardCount  uint64
	index       uint64
	indexLock   *sync.RWMutex
	subscribers map[string]map[uint64]bool
	subLock     *sync.RWMutex
	subBuffer   map[uint64]*shard
}

// New - Create a new Pub/Sub hub, using which messages
// can be routed to various topics
func New(shardCount uint64) *PubSub {
	broker := &PubSub{
		shardCount:  shardCount,
		index:       1,
		indexLock:   &sync.RWMutex{},
		subscribers: make(map[string]map[uint64]bool),
		subLock:     &sync.RWMutex{},
		subBuffer:   make(map[uint64]*shard),
	}

	var i uint64 = 0
	for ; i < broker.shardCount; i++ {
		broker.subBuffer[i] = &shard{lock: &sync.RWMutex{}, subscribers: make(map[uint64]*subscriberInfo)}
	}

	return broker
}

// Publish - Publish message to N-topics in concurrent-safe manner
func (p *PubSub) Publish(msg *Message) uint64 {
	var c uint64
	var mLen = len(msg.Data)

	for i := 0; i < len(msg.Topics); i++ {
		topic := msg.Topics[i]

		p.subLock.RLock()
		subs, ok := p.subscribers[topic]
		if ok && len(subs) != 0 {

			for id := range subs {
				shard, ok := p.subBuffer[id%p.shardCount]
				if !ok {
					continue
				}

				shard.lock.RLock()
				sub, ok := shard.subscribers[id]
				shard.lock.RUnlock()
				if !ok {
					continue
				}

				buf := make([]byte, mLen)
				n := copy(buf, msg.Data)
				if n != mLen {
					continue
				}

				// -- Concurrent-safe append, starts
				sub.lock.Lock()
				bufLen := len(sub.buffer)
				bufCap := cap(sub.buffer)

				if bufLen+1 > bufCap {

					bufTmp := make([]*PublishedMessage, bufLen+1)
					n := copy(bufTmp[:], sub.buffer[:])
					if n != bufLen {
						sub.lock.Unlock()
						continue
					}

					old := sub.buffer
					sub.buffer = bufTmp
					pMsg := PublishedMessage{Topic: topic, Data: buf}
					copy(sub.buffer[bufLen:], []*PublishedMessage{&pMsg})

					// Previous slice being zeroed
					for i := 0; i < bufLen; i++ {
						old[i] = nil
					}

				} else {
					pMsg := PublishedMessage{Topic: topic, Data: buf}
					sub.buffer = append(sub.buffer, &pMsg)
				}

				sub.lock.Unlock()
				// -- Concurrent-safe append, ends

				if len(sub.ping) < cap(sub.ping) {
					sub.ping <- struct{}{}
				}

				c++
			}

		}
		p.subLock.RUnlock()

	}

	return c
}

// Subscribe - Create new subscriber instance with initial capacity,
// listening for messages published on N-topics initially.
//
// More topics can be subscribed to later using returned subscriber instance.
func (p *PubSub) Subscribe(cap int, topics ...string) *Subscriber {
	if len(topics) == 0 {
		return nil
	}

	sub := &Subscriber{
		id: p.nextId(),
		info: &subscriberInfo{
			ping:   make(chan struct{}, cap),
			lock:   &sync.RWMutex{},
			buffer: make([]*PublishedMessage, 0, cap),
		},
		tLock:  &sync.RWMutex{},
		topics: make(map[string]bool),
		hub:    p,
	}

	for i := 0; i < len(topics); i++ {
		sub.topics[topics[i]] = true
	}

	p.addSubscription(&subscriptionRequest{
		id:     sub.id,
		topics: topics,
	})

	shard := p.subBuffer[sub.id%p.shardCount]
	shard.lock.Lock()
	shard.subscribers[sub.id] = sub.info
	shard.lock.Unlock()

	return sub
}

func (p *PubSub) nextId() uint64 {
	p.indexLock.Lock()
	defer p.indexLock.Unlock()

	id := p.index
	p.index++
	return id
}

func (p *PubSub) addSubscription(req *subscriptionRequest) uint64 {
	var c uint64

	for i := 0; i < len(req.topics); i++ {
		p.subLock.Lock()
		subs, ok := p.subscribers[req.topics[i]]
		if !ok {
			subs = make(map[uint64]bool)
		}
		if _, ok := subs[req.id]; !ok {
			subs[req.id] = true
			c++
		}
		p.subscribers[req.topics[i]] = subs
		p.subLock.Unlock()
	}

	return c
}

func (p *PubSub) unsubscribe(req *unsubscriptionRequest) uint64 {
	var c uint64

	for i := 0; i < len(req.topics); i++ {
		p.subLock.Lock()
		if subs, ok := p.subscribers[req.topics[i]]; ok {
			if _, ok := subs[req.id]; ok {
				delete(subs, req.id)
				c++
			}

			if len(subs) == 0 {
				delete(p.subscribers, req.topics[i])
			}
		}
		p.subLock.Unlock()
	}

	return c
}

func (p *PubSub) destroy(id uint64) {
	shard, ok := p.subBuffer[id%p.shardCount]
	if !ok {
		return
	}

	shard.lock.Lock()
	defer shard.lock.Unlock()

	delete(shard.subscribers, id)
}
