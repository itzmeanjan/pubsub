package pubsub

import (
	"context"
	"io"
	"sync"
)

// PubSub - Pub/Sub Server i.e. holds which clients are subscribed to what topics,
// manages publishing messages to correct topics, handles (un-)subscription requests
//
// In other words state manager of Pub/Sub system
type PubSub struct {
	SafetyMode       bool
	Alive            bool
	Index            uint64
	MessageChan      chan *PublishRequest
	SubscriberIdChan chan chan uint64
	SubscribeChan    chan *SubscriptionRequest
	UnsubscribeChan  chan *UnsubscriptionRequest
	Subscribers      map[string]map[uint64]io.Writer
}

// New - Create a new Pub/Sub hub, using which messages
// can be routed to various topics
func New() *PubSub {
	return &PubSub{
		Alive:            false,
		Index:            1,
		MessageChan:      make(chan *PublishRequest, 1),
		SubscriberIdChan: make(chan chan uint64, 1),
		SubscribeChan:    make(chan *SubscriptionRequest, 1),
		UnsubscribeChan:  make(chan *UnsubscriptionRequest, 1),
		Subscribers:      make(map[string]map[uint64]io.Writer),
	}
}

// Start - Handles request from publishers & subscribers, so that
// message publishing can be abstracted
//
// Consider running it as a go routine
func (p *PubSub) Start(ctx context.Context) {

	// Because pub/sub system is now running
	// & it's ready to process requests
	p.Alive = true

	for {

		select {

		case <-ctx.Done():
			p.Alive = false
			return

		case req := <-p.MessageChan:

			var publishedOn uint64
			var writers = make([]io.Writer, 0)
			var _writers = make(map[io.Writer]bool)

			for i := 0; i < len(req.Message.Topics); i++ {
				topic := req.Message.Topics[i]
				if subs, ok := p.Subscribers[topic]; ok {
					for _, w := range subs {
						if _, ok := _writers[w]; ok {
							continue
						}

						writers = append(writers, w)
						_writers[w] = true
						publishedOn++
					}
				}
			}

			if publishedOn != 0 {
				w := io.MultiWriter(writers...)
				req.Message.Data.WriteTo(w)
			}
			req.ResponseChan <- publishedOn

		case req := <-p.SubscriberIdChan:

			req <- p.Index
			p.Index++

		case req := <-p.SubscribeChan:

			var subscribedTo uint64

			for i := 0; i < len(req.Topics); i++ {
				topic := req.Topics[i]
				subs, ok := p.Subscribers[topic]
				if !ok {
					p.Subscribers[topic] = make(map[uint64]io.Writer)
					p.Subscribers[topic][req.Id] = req.Writer
					subscribedTo++

					continue
				}

				if _, ok := subs[req.Id]; !ok {
					subs[req.Id] = req.Writer
					subscribedTo++
				}
			}

			req.ResponseChan <- subscribedTo

		case req := <-p.UnsubscribeChan:

			var unsubscribedFrom uint64

			for i := 0; i < len(req.Topics); i++ {
				topic := req.Topics[i]
				if subs, ok := p.Subscribers[topic]; ok {
					if _, ok := subs[req.Id]; ok {
						delete(subs, req.Id)
						unsubscribedFrom++
					}

					if len(subs) == 0 {
						delete(p.Subscribers, topic)
					}
				}
			}

			req.ResponseChan <- unsubscribedFrom

		}

	}

}

func (p *PubSub) Publish(msg *Message) (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		p.MessageChan <- &PublishRequest{Message: msg, ResponseChan: resChan}

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) Subscribe(cap int, topics ...string) *Subscriber {
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
			Id:     id,
			Reader: r,
			Writer: w,
			mLock:  &sync.RWMutex{},
			tLock:  &sync.RWMutex{},
			Buffer: make([]*PublishedMessage, 0, cap),
			Topics: make(map[string]bool),
			hub:    p,
		}

		for i := 0; i < len(topics); i++ {
			sub.Topics[topics[i]] = true
		}

		resChan := make(chan uint64)
		p.SubscribeChan <- &SubscriptionRequest{
			Id:           sub.Id,
			Writer:       sub.Writer,
			Topics:       topics,
			ResponseChan: resChan,
		}
		<-resChan

		return sub
	}

	return nil
}

func (p *PubSub) nextId() (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		p.SubscriberIdChan <- resChan

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) addSubscription(subReq *SubscriptionRequest) (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		subReq.ResponseChan = resChan
		p.SubscribeChan <- subReq

		return true, <-resChan
	}

	return false, 0
}

func (p *PubSub) unsubscribe(unsubReq *UnsubscriptionRequest) (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		unsubReq.ResponseChan = resChan
		p.UnsubscribeChan <- unsubReq

		return true, <-resChan
	}

	return false, 0
}
