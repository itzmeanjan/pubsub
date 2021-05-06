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
	SafetyChan       chan *SafetyMode
	Subscribers      map[string]map[uint64]io.Writer
}

// New - Create a new Pub/Sub hub, using which messages
// can be routed to various topics
func New() *PubSub {
	return &PubSub{
		SafetyMode:       true,
		Alive:            false,
		Index:            1,
		MessageChan:      make(chan *PublishRequest, 1),
		SubscriberIdChan: make(chan chan uint64, 1),
		SubscribeChan:    make(chan *SubscriptionRequest, 1),
		UnsubscribeChan:  make(chan *UnsubscriptionRequest, 1),
		SafetyChan:       make(chan *SafetyMode, 1),
		Subscribers:      make(map[string]map[uint64]io.Writer),
	}
}

// AllowUnsafe - Hub allows you to pass slice of messages to N-many
// topic subscribers & as slices are references if any of those subscribers
// ( or even publisher itself ) mutates slice it'll be reflected to
// all parties, which might not be desireable always.
//
// But if you're sure that won't cause any problem for you,
// you can at your own risk disable SAFETY lock
//
// If disabled, hub won't anymore attempt to copy slices to
// for each topic subscriber, it'll simply pass. As this means
// hub will do lesser work, hub will be able to process more
// data than ever **FASTer ⭐️**
//
// ❗️ But remember this might bring problems for you
func (p *PubSub) AllowUnsafe() bool {
	if p.Alive {
		resChan := make(chan bool)
		p.SafetyChan <- &SafetyMode{Enable: false, ResponseChan: resChan}

		return <-resChan
	}

	return false
}

// OnlySafe - You'll probably never require to use this method
// if you've not explicitly disabled safety lock by invoking `AllowUnsafe` ( 👆)
//
// But you've & in runtime you need to again enable safety mode, you can call this
// method & all messages published are going to be copied for each subscriber
// which will make ops slower that SAFETY lock disabled mode
func (p *PubSub) OnlySafe() bool {
	if p.Alive {
		resChan := make(chan bool)
		p.SafetyChan <- &SafetyMode{Enable: true, ResponseChan: resChan}

		return <-resChan
	}

	return false
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
			var writers = make([]io.Writer, 0, 1)
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

		case req := <-p.SafetyChan:

			p.SafetyMode = req.Enable
			req.ResponseChan <- true

		}

	}

}

// Publish - Publish message to N-many topics, receives how many of
// subscribers are receiving ( will receive ) copy of this message
//
// Response will only be negative if Pub/Sub system has stopped running
func (p *PubSub) Publish(msg *Message) (bool, uint64) {

	if p.Alive {
		resChan := make(chan uint64)
		p.MessageChan <- &PublishRequest{Message: msg, ResponseChan: resChan}

		return true, <-resChan
	}

	return false, 0

}

func (p *PubSub) nextId() uint64 {
	resChan := make(chan uint64)
	p.SubscriberIdChan <- resChan

	return <-resChan
}

func (p *PubSub) Subscribe(cap int, topics ...string) *Subscriber {

	if p.Alive {
		if len(topics) == 0 {
			return nil
		}

		r, w := io.Pipe()
		sub := &Subscriber{
			Id:     p.nextId(),
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

func (p *PubSub) addSubscription(subReq *SubscriptionRequest) (bool, uint64) {
	if p.Alive {
		resChan := make(chan uint64)
		subReq.ResponseChan = resChan
		p.SubscribeChan <- subReq

		return true, <-resChan
	}

	return false, 0
}

// AddSubscription - Use existing subscriber client to subscribe to more topics
//
// Response will only be negative if Pub/Sub system has stopped running
func (p *PubSub) AddSubscription(subscriber *Subscriber, topics ...string) (bool, uint64) {

	if p.Alive {
		if len(topics) == 0 {
			return true, 0
		}

		for i := 0; i < len(topics); i++ {
			if state, ok := subscriber.Topics[topics[i]]; ok && state {
				continue
			}

			subscriber.Topics[topics[i]] = true
		}

		resChan := make(chan uint64)
		p.SubscribeChan <- &SubscriptionRequest{
			Id:           subscriber.Id,
			Writer:       subscriber.Writer,
			Topics:       topics,
			ResponseChan: resChan,
		}
		return true, <-resChan
	}

	return false, 0

}

// Unsubscribe - Unsubscribes from topics for specified subscriber client
//
// Response will only be negative if Pub/Sub system has stopped running
func (p *PubSub) Unsubscribe(subscriber *Subscriber, topics ...string) (bool, uint64) {

	if p.Alive {

		if len(topics) == 0 {
			return true, 0
		}

		_topics := make([]string, 0, len(topics))
		for i := 0; i < len(topics); i++ {

			if state, ok := subscriber.Topics[topics[i]]; ok {
				if state {
					_topics = append(_topics, topics[i])
					subscriber.Topics[topics[i]] = false
				}
			}

		}

		if len(_topics) == 0 {
			return true, 0
		}

		resChan := make(chan uint64)
		p.UnsubscribeChan <- &UnsubscriptionRequest{
			Id:           subscriber.Id,
			Topics:       _topics,
			ResponseChan: resChan,
		}

		return true, <-resChan

	}

	return false, 0

}

// UnsubscribeAll - All current active subscriptions get unsubscribed from
//
// Response will only be negative if Pub/Sub system has stopped running
func (p *PubSub) UnsubscribeAll(subscriber *Subscriber) (bool, uint64) {

	if p.Alive {
		topics := make([]string, 0, len(subscriber.Topics))
		for topic, ok := range subscriber.Topics {
			if ok {
				topics = append(topics, topic)
				subscriber.Topics[topic] = false
			}
		}

		if len(topics) == 0 {
			return true, 0
		}

		resChan := make(chan uint64)
		p.UnsubscribeChan <- &UnsubscriptionRequest{
			Id:           subscriber.Id,
			Topics:       topics,
			ResponseChan: resChan,
		}

		return true, <-resChan
	}

	return false, 0

}
