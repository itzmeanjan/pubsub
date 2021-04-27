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
	SubscribeChan    chan *SubscriptionRequest
	UnsubscribeChan  chan *UnsubscriptionRequest
	Subscribers      map[string]map[uint64]chan *PublishedMessage
}

func New() *PubSub {
	return &PubSub{
		Alive:            true,
		Index:            1,
		MessageChan:      make(chan *PublishRequest, 1),
		SubscriberIdChan: make(chan chan uint64, 1),
		SubscribeChan:    make(chan *SubscriptionRequest, 1),
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

		case req := <-p.SubscribeChan:

			var subscribedTo uint64

			for topic := range req.Subscriber.Topics {

				subs, ok := p.Subscribers[topic]
				if !ok {

					p.Subscribers[topic] = make(map[uint64]chan *PublishedMessage)
					p.Subscribers[topic][req.Subscriber.Id] = req.Subscriber.Channel
					subscribedTo++

					continue
				}

				if _, ok := subs[req.Subscriber.Id]; !ok {
					subs[req.Subscriber.Id] = req.Subscriber.Channel
					subscribedTo++
				}

			}

			req.ResponseChan <- subscribedTo

		case req := <-p.UnsubscribeChan:

			var unsubscribedFrom uint64

			for i := 0; i < len(req.Topics); i++ {

				if subs, ok := p.Subscribers[req.Topics[i]]; ok {

					if _, ok := subs[req.Id]; ok {
						delete(subs, req.Id)
						unsubscribedFrom++
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

		resChan := make(chan uint64)
		p.SubscribeChan <- &SubscriptionRequest{Subscriber: sub, ResponseChan: resChan}
		<-resChan

		return sub

	}

	return nil

}

func (p *PubSub) AddSubscription(subscriber *Subscriber, topics ...string) (bool, uint64) {

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

		if len(_subscriber.Topics) == 0 {
			return true, 0
		}

		resChan := make(chan uint64)
		p.SubscribeChan <- &SubscriptionRequest{Subscriber: _subscriber, ResponseChan: resChan}

		return true, <-resChan

	}

	return false, 0

}

func (p *PubSub) Unsubscribe(subscriber *Subscriber, topics ...string) (bool, uint64) {

	if p.Alive {

		_topics := make([]string, 0, len(topics))
		for i := 0; i < len(_topics); i++ {

			if state, ok := subscriber.Topics[topics[i]]; ok {
				if state {
					_topics = append(_topics, topics[i])
					subscriber.Topics[topics[i]] = false
				}
			}

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

func (p *PubSub) UnsubscribeAll(subscriber *Subscriber) (bool, uint64) {

	if p.Alive {

		topics := make([]string, len(subscriber.Topics))
		for topic := range subscriber.Topics {
			topics = append(topics, topic)
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
