package pubsub

import "context"

type Message struct {
	Topics []string
	Data   []byte
}

type PublishRequest struct {
	Message      *Message
	ResponseChan chan uint64
}

type PubSub struct {
	Alive            bool
	Index            uint64
	MessageChan      chan *PublishRequest
	SubscriberIdChan chan chan uint64
	SubscribeChan    chan *Subscriber
	UnsubscribeChan  chan *UnsubscriptionRequest
	Subscribers      map[string]map[uint64]chan *PublishedMessage
}

func New() *PubSub {
	return &PubSub{
		Alive:            true,
		Index:            1,
		MessageChan:      make(chan *PublishRequest, 1),
		SubscriberIdChan: make(chan chan uint64, 1),
		SubscribeChan:    make(chan *Subscriber, 1),
		UnsubscribeChan:  make(chan *UnsubscriptionRequest, 1),
		Subscribers:      make(map[string]map[uint64]chan *PublishedMessage),
	}
}

func (p *PubSub) Start(ctx context.Context) {

	for {

		select {

		case <-ctx.Done():
			p.Alive = false
			return

		case req := <-p.MessageChan:

			var publishedOn uint64

			for i := 0; i < len(req.Message.Topics); i++ {

				topic := req.Message.Topics[i]
				subs, ok := p.Subscribers[topic]
				if !ok {
					continue
				}

				for _, sub := range subs {

					msg := PublishedMessage{
						Data:  req.Message.Data,
						Topic: topic,
					}

					if len(sub) < cap(sub) {
						sub <- &msg
						publishedOn++
					}

				}

			}

			req.ResponseChan <- publishedOn

		case req := <-p.SubscriberIdChan:

			req <- p.Index
			p.Index++

		case sub := <-p.SubscribeChan:

			for topic := range sub.Topics {

				subs, ok := p.Subscribers[topic]
				if !ok {
					p.Subscribers[topic] = make(map[uint64]chan *PublishedMessage)
					p.Subscribers[topic][sub.Id] = sub.Channel
					continue
				}

				if _, ok := subs[sub.Id]; !ok {
					subs[sub.Id] = sub.Channel
				}

			}

		case req := <-p.UnsubscribeChan:

			for i := 0; i < len(req.Topics); i++ {

				if subs, ok := p.Subscribers[req.Topics[i]]; ok {
					delete(subs, req.Id)
				}

			}

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

func (p *PubSub) Subscribe(cap uint64, topics ...string) *Subscriber {

	if p.Alive {

		idGenChan := make(chan uint64)
		p.SubscriberIdChan <- idGenChan

		sub := &Subscriber{
			Id:      <-idGenChan,
			Channel: make(chan *PublishedMessage, cap),
			Topics:  make(map[string]bool),
		}

		for i := 0; i < len(topics); i++ {
			sub.Topics[topics[i]] = true
		}

		p.SubscribeChan <- sub
		return sub

	}

	return nil

}

func (p *PubSub) AddSubscription(cap uint64, subscriber *Subscriber, topics ...string) bool {

	if p.Alive {

		_subscriber := &Subscriber{
			Id:      subscriber.Id,
			Channel: subscriber.Channel,
			Topics:  make(map[string]bool),
		}

		for i := 0; i < len(topics); i++ {

			if state, ok := subscriber.Topics[topics[i]]; ok {
				if state {
					continue
				}
			}

			_subscriber.Topics[topics[i]] = true

		}

		p.SubscribeChan <- _subscriber
		return true

	}

	return false

}

func (p *PubSub) Unsubscribe(subscriber *Subscriber, topics ...string) bool {

	if p.Alive {
		p.UnsubscribeChan <- &UnsubscriptionRequest{Id: subscriber.Id, Topics: topics}
		return true
	}

	return false

}
