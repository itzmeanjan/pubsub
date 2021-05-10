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
	alive           bool
	aliveLock       *sync.RWMutex
	index           uint64
	indexLock       *sync.RWMutex
	messageChan     chan *publishRequest
	subscribeChan   chan *subscriptionRequest
	unsubscribeChan chan *unsubscriptionRequest
	destroyChan     chan *destroyRequest
	subscribers     map[string]map[uint64]bool
	subBuffer       map[uint64]*subscriberInfo
}

// New - Create a new Pub/Sub hub, using which messages
// can be routed to various topics
func New(ctx context.Context) *PubSub {
	broker := &PubSub{
		alive:           false,
		aliveLock:       &sync.RWMutex{},
		index:           1,
		indexLock:       &sync.RWMutex{},
		messageChan:     make(chan *publishRequest, 1),
		subscribeChan:   make(chan *subscriptionRequest, 1),
		unsubscribeChan: make(chan *unsubscriptionRequest, 1),
		destroyChan:     make(chan *destroyRequest, 1),
		subscribers:     make(map[string]map[uint64]bool),
		subBuffer:       make(map[uint64]*subscriberInfo),
	}

	started := make(chan struct{})
	go broker.start(ctx, started)
	<-started

	return broker
}

// Publish - Send message publishing request to N-topics in concurrent-safe manner
func (p *PubSub) Publish(msg *Message) (bool, uint64) {
	if p.IsAlive() {
		resChan := make(chan uint64)
		p.messageChan <- &publishRequest{message: msg, responseChan: resChan}

		return true, <-resChan
	}

	return false, 0
}

// Subscribe - Create new subscriber instance with initial buffer capacity,
// listening for messages published on N-topics initially.
//
// More topics can be subscribed to later using returned subscriber instance.
func (p *PubSub) Subscribe(ctx context.Context, cap int, topics ...string) *Subscriber {
	if p.IsAlive() {
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

		resChan := make(chan uint64)
		p.subscribeChan <- &subscriptionRequest{
			id:           sub.id,
			info:         sub.info,
			topics:       topics,
			responseChan: resChan,
		}
		<-resChan

		return sub
	}

	return nil
}

// start - Handles request from publishers & subscribers, so that
// message publishing can be abstracted
//
// Consider running it as a go routine
func (p *PubSub) start(ctx context.Context, started chan struct{}) {

	// Because pub/sub system is now running
	// & it's ready to process requests
	p.toggleState()
	close(started)

	for {
		select {

		case <-ctx.Done():
			p.toggleState()
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

		case req := <-p.subscribeChan:
			var subscribedTo uint64

			if _, ok := p.subBuffer[req.id]; !ok {
				p.subBuffer[req.id] = req.info
			}

			for i := 0; i < len(req.topics); i++ {
				topic := req.topics[i]
				subs, ok := p.subscribers[topic]
				if !ok {
					p.subscribers[topic] = make(map[uint64]bool)
					p.subscribers[topic][req.id] = true
					subscribedTo++

					continue
				}

				if _, ok := subs[req.id]; !ok {
					subs[req.id] = true
					subscribedTo++
				}
			}

			req.responseChan <- subscribedTo

		case req := <-p.unsubscribeChan:
			var unsubscribedFrom uint64

			for i := 0; i < len(req.topics); i++ {
				topic := req.topics[i]
				if subs, ok := p.subscribers[topic]; ok {
					if _, ok := subs[req.id]; ok {
						delete(subs, req.id)
						unsubscribedFrom++
					}

					if len(subs) == 0 {
						delete(p.subscribers, topic)
					}
				}
			}

			req.responseChan <- unsubscribedFrom

		case req := <-p.destroyChan:

			delete(p.subBuffer, req.id)
			req.repsonseChan <- true

		}
	}

}

// IsAlive - Check whether Hub is still alive or not [ concurrent-safe, good for external usage ]
func (p *PubSub) IsAlive() bool {
	p.aliveLock.RLock()
	defer p.aliveLock.RUnlock()

	return p.alive
}

// toggleState - Marks hub enabled/ disabled [ concurrent-safe, internal usage ]
func (p *PubSub) toggleState() {
	p.aliveLock.Lock()
	defer p.aliveLock.Unlock()

	p.alive = !p.alive
}

func (p *PubSub) nextId() uint64 {
	p.indexLock.Lock()
	defer p.indexLock.Unlock()

	id := p.index
	p.index++
	return id
}

func (p *PubSub) addSubscription(subReq *subscriptionRequest) (bool, uint64) {
	if p.IsAlive() {
		resChan := make(chan uint64)
		subReq.responseChan = resChan
		p.subscribeChan <- subReq

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) unsubscribe(unsubReq *unsubscriptionRequest) (bool, uint64) {
	if p.IsAlive() {
		resChan := make(chan uint64)
		unsubReq.responseChan = resChan
		p.unsubscribeChan <- unsubReq

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) destroy(destroyReq *destroyRequest) bool {
	if p.IsAlive() {
		resChan := make(chan bool)
		destroyReq.repsonseChan = resChan
		p.destroyChan <- destroyReq

		return <-resChan
	}

	return false
}
