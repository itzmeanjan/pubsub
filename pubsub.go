package pubsub

import (
	"context"
	"io"
	"log"
	"sync"
)

// PubSub - Pub/Sub Server i.e. holds which clients are subscribed to what topics,
// manages publishing messages to correct topics, handles (un-)subscription requests
//
// In other words state manager of Pub/Sub Broker
type PubSub struct {
	Alive            bool
	index            uint64
	messageChan      chan *publishRequest
	subscriberIdChan chan chan uint64
	subscribeChan    chan *subscriptionRequest
	unsubscribeChan  chan *unsubscriptionRequest
	subscribers      map[string]map[uint64]*subscriberInfo
}

// New - Create a new Pub/Sub hub, using which messages
// can be routed to various topics
func New(ctx context.Context) *PubSub {
	broker := &PubSub{
		Alive:            false,
		index:            1,
		messageChan:      make(chan *publishRequest, 1),
		subscriberIdChan: make(chan chan uint64, 1),
		subscribeChan:    make(chan *subscriptionRequest, 1),
		unsubscribeChan:  make(chan *unsubscriptionRequest, 1),
		subscribers:      make(map[string]map[uint64]*subscriberInfo),
	}

	started := make(chan struct{})
	go broker.start(ctx, started)
	<-started

	return broker
}

// Publish - Send message publishing request to N-topics in concurrent-safe manner
func (p *PubSub) Publish(msg *Message) (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		p.messageChan <- &publishRequest{Message: msg, ResponseChan: resChan}

		return true, <-resChan
	}

	return false, 0
}

// Subscribe - Create new subscriber instance with initial buffer capacity,
// listening for messages published on N-topics initially.
//
// More topics can be subscribed to later using returned subscriber instance.
func (p *PubSub) Subscribe(ctx context.Context, cap int, topics ...string) *Subscriber {
	if p.Alive {
		if len(topics) == 0 {
			return nil
		}
		ok, id := p.nextId()
		if !ok {
			return nil
		}
		r, w := io.Pipe()

		sub := &Subscriber{
			id:     id,
			reader: r,
			info: &subscriberInfo{
				Writer: w,
				Ping:   make(chan struct{}, cap),
			},
			mLock:  &sync.RWMutex{},
			tLock:  &sync.RWMutex{},
			topics: make(map[string]bool),
			buffer: make([]*PublishedMessage, 0, cap),
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

		started := make(chan struct{})
		go sub.start(ctx, started)
		<-resChan
		<-started

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
	p.Alive = true
	close(started)

	for {
		select {

		case <-ctx.Done():
			p.Alive = false
			return

		case req := <-p.messageChan:
			var publishedOn uint64

			for i := 0; i < len(req.Message.Topics); i++ {
				topic := req.Message.Topics[i]

				if subs, ok := p.subscribers[topic.String()]; ok {
					writers := make([]io.Writer, 0, 1)

					for _, w := range subs {
						w.Ping <- struct{}{}
						writers = append(writers, w.Writer)

						publishedOn++
					}

					if len(writers) != 0 {
						w := io.MultiWriter(writers...)

						msg := PublishedMessage{
							Topic: topic,
							Data:  req.Message.Data,
						}
						if _, err := msg.WriteTo(w); err != nil {
							log.Printf("[pubsub] Error : %s\n", err.Error())
							continue
						}

					}
				}
			}

			req.ResponseChan <- publishedOn

		case req := <-p.subscriberIdChan:
			req <- p.index
			p.index++

		case req := <-p.subscribeChan:
			var subscribedTo uint64

			for i := 0; i < len(req.Topics); i++ {
				topic := req.Topics[i]
				subs, ok := p.subscribers[topic]
				if !ok {
					p.subscribers[topic] = make(map[uint64]*subscriberInfo)
					p.subscribers[topic][req.Id] = req.info
					subscribedTo++

					continue
				}

				if _, ok := subs[req.Id]; !ok {
					subs[req.Id] = req.info
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

func (p *PubSub) nextId() (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		p.subscriberIdChan <- resChan

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) addSubscription(subReq *subscriptionRequest) (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		subReq.ResponseChan = resChan
		p.subscribeChan <- subReq

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) unsubscribe(unsubReq *unsubscriptionRequest) (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		unsubReq.ResponseChan = resChan
		p.unsubscribeChan <- unsubReq

		return true, <-resChan
	}

	return false, 0
}
