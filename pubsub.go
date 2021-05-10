package pubsub

import (
	"context"
	"sync"
)

// PubSub - Pub/Sub Server i.e. holds which clients are subscribed to what topics,
// manages publishing messages to correct topics, handles (un-)subscription requests
//
// In other words state manager of Pub/Sub Broker
type PubSub struct {
	index         uint64
	indexLock     *sync.RWMutex
	messageChan   chan *publishRequest
	subscribers   map[string]map[uint64]bool
	subLock       *sync.RWMutex
	subBuffer     map[uint64]*subscriberInfo
	subBufferLock *sync.RWMutex
}

// New - Create a new Pub/Sub hub, using which messages
// can be routed to various topics
func New(ctx context.Context) *PubSub {
	broker := &PubSub{
		index:         1,
		indexLock:     &sync.RWMutex{},
		messageChan:   make(chan *publishRequest, 1),
		subscribers:   make(map[string]map[uint64]bool),
		subLock:       &sync.RWMutex{},
		subBuffer:     make(map[uint64]*subscriberInfo),
		subBufferLock: &sync.RWMutex{},
	}

	started := make(chan struct{})
	go broker.start(ctx, started)
	<-started

	return broker
}

// Publish - Send message publishing request to N-topics in concurrent-safe manner
func (p *PubSub) Publish(msg *Message) (bool, uint64) {
	resChan := make(chan uint64)
	p.messageChan <- &publishRequest{message: msg, responseChan: resChan}

	return true, <-resChan
}

// Subscribe - Create new subscriber instance with initial capacity,
// listening for messages published on N-topics initially.
//
// More topics can be subscribed to later using returned subscriber instance.
func (p *PubSub) Subscribe(ctx context.Context, cap int, topics ...string) *Subscriber {
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

	p.subBufferLock.Lock()
	p.subBuffer[sub.id] = sub.info
	p.subBufferLock.Unlock()

	return sub
}

// start - Handles request from publishers & subscribers, so that
// message publishing can be abstracted
//
// Consider running it as a go routine
func (p *PubSub) start(ctx context.Context, started chan struct{}) {

	// Because pub/sub system is now running
	// & it's ready to process requests
	close(started)

	for {
		select {

		case <-ctx.Done():
			return

		case req := <-p.messageChan:
			var publishedOn uint64
			var msgLen = len(req.message.Data)

			for i := 0; i < len(req.message.Topics); i++ {
				topic := req.message.Topics[i]

				if subs, ok := p.subscribers[topic]; ok && len(subs) != 0 {

					for id := range subs {
						sub, ok := p.subBuffer[id]
						if !ok {
							continue
						}

						buf := make([]byte, msgLen)
						n := copy(buf, req.message.Data)
						if n != msgLen {
							continue
						}

						sub.lock.Lock()
						sub.buffer = append(sub.buffer, &PublishedMessage{Topic: topic, Data: buf})
						sub.lock.Unlock()

						if len(sub.ping) < cap(sub.ping) {
							sub.ping <- struct{}{}
						}

						publishedOn++
					}

				}
			}

			req.responseChan <- publishedOn

		}
	}

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
	p.subBufferLock.Lock()
	defer p.subBufferLock.Unlock()

	delete(p.subBuffer, id)
}
