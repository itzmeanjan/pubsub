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
	alive            bool
	lock             *sync.RWMutex
	index            uint64
	pubMessageChan   chan *publishRequest
	conMessageChan   chan *consumptionRequest
	subscriberIdChan chan chan uint64
	subscribeChan    chan *subscriptionRequest
	unsubscribeChan  chan *unsubscriptionRequest
	subscribers      map[string]map[uint64]bool
	subBuffer        map[uint64]*subscriberInfo
}

// New - Create a new Pub/Sub hub, using which messages
// can be routed to various topics
func New(ctx context.Context) *PubSub {
	broker := &PubSub{
		alive:            false,
		lock:             &sync.RWMutex{},
		index:            1,
		pubMessageChan:   make(chan *publishRequest, 1),
		conMessageChan:   make(chan *consumptionRequest, 1),
		subscriberIdChan: make(chan chan uint64, 1),
		subscribeChan:    make(chan *subscriptionRequest, 1),
		unsubscribeChan:  make(chan *unsubscriptionRequest, 1),
		subscribers:      make(map[string]map[uint64]bool),
		subBuffer:        make(map[uint64]*subscriberInfo),
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
		p.pubMessageChan <- &publishRequest{Message: msg, ResponseChan: resChan}

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
		ok, id := p.nextId()
		if !ok {
			return nil
		}

		sub := &Subscriber{
			id: id,
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
			Id:           sub.id,
			info:         sub.info,
			Topics:       topics,
			ResponseChan: resChan,
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

		case req := <-p.pubMessageChan:
			var publishedOn uint64
			var msgLen = len(req.Message.Data)

			for i := 0; i < len(req.Message.Topics); i++ {
				topic := req.Message.Topics[i]

				if subs, ok := p.subscribers[topic.String()]; ok && len(subs) != 0 {

					for id := range subs {
						buf := make([]byte, msgLen)
						n := copy(buf, req.Message.Data)
						if n != msgLen {
							continue
						}

						msg := PublishedMessage{Topic: topic, Data: buf}
						sub, ok := p.subBuffer[id]
						if !ok {
							continue
						}

						sub.lock.Lock()
						sub.buffer = append(sub.buffer, &msg)
						sub.lock.Unlock()

						if len(sub.ping) < cap(sub.ping) {
							sub.ping <- struct{}{}
						}

						publishedOn++
					}

				}
			}

			req.ResponseChan <- publishedOn

		case req := <-p.conMessageChan:

			sub, ok := p.subBuffer[req.Id]
			if !ok {
				req.ResponseChan <- false
				break
			}

			sub.lock.RLock()
			req.ResponseChan <- len(sub.buffer) > 0
			sub.lock.RUnlock()

		case req := <-p.subscriberIdChan:
			req <- p.index
			p.index++

		case req := <-p.subscribeChan:
			var subscribedTo uint64

			if _, ok := p.subBuffer[req.Id]; !ok {
				p.subBuffer[req.Id] = req.info
			}

			for i := 0; i < len(req.Topics); i++ {
				topic := req.Topics[i]
				subs, ok := p.subscribers[topic]
				if !ok {
					p.subscribers[topic] = make(map[uint64]bool)
					p.subscribers[topic][req.Id] = true
					subscribedTo++

					continue
				}

				if _, ok := subs[req.Id]; !ok {
					subs[req.Id] = true
					subscribedTo++
				}
			}

			req.ResponseChan <- subscribedTo

		case req := <-p.unsubscribeChan:
			var unsubscribedFrom uint64

			for i := 0; i < len(req.Topics); i++ {
				topic := req.Topics[i]
				if subs, ok := p.subscribers[topic]; ok {
					if _, ok := subs[req.Id]; ok {
						delete(subs, req.Id)
						unsubscribedFrom++
					}

					if len(subs) == 0 {
						delete(p.subscribers, topic)
					}
				}
			}

			req.ResponseChan <- unsubscribedFrom

		}
	}

}

// IsAlive - Check whether Hub is still alive or not [ concurrent-safe, good for external usage ]
func (p *PubSub) IsAlive() bool {
	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.alive
}

// toggleState - Marks hub enabled/ disabled [ concurrent-safe, internal usage ]
func (p *PubSub) toggleState() {
	p.lock.Lock()
	defer p.lock.Unlock()

	p.alive = !p.alive
}

func (p *PubSub) nextId() (bool, uint64) {
	if p.IsAlive() {
		resChan := make(chan uint64)
		p.subscriberIdChan <- resChan

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) addSubscription(subReq *subscriptionRequest) (bool, uint64) {
	if p.IsAlive() {
		resChan := make(chan uint64)
		subReq.ResponseChan = resChan
		p.subscribeChan <- subReq

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) unsubscribe(unsubReq *unsubscriptionRequest) (bool, uint64) {
	if p.IsAlive() {
		resChan := make(chan uint64)
		unsubReq.ResponseChan = resChan
		p.unsubscribeChan <- unsubReq

		return true, <-resChan
	}

	return false, 0
}
