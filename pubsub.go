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
	Alive         bool
	MessageChan   chan *PublishRequest
	SubscribeChan chan *Subscriber
	Subscribers   map[string]map[*Subscriber]bool
}

func New() *PubSub {
	return &PubSub{
		Alive:         true,
		MessageChan:   make(chan *PublishRequest, 1),
		SubscribeChan: make(chan *Subscriber, 1),
		Subscribers:   make(map[string]map[*Subscriber]bool),
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

				subs, ok := p.Subscribers[req.Message.Topics[i]]
				if !ok {
					continue
				}

				for sub := range subs {
					sub.Channel <- req.Message.Data
					publishedOn++
				}

			}

			req.ResponseChan <- publishedOn

		case sub := <-p.SubscribeChan:

			for topic := range sub.Topics {

				subs, ok := p.Subscribers[topic]
				if !ok {
					p.Subscribers[topic] = make(map[*Subscriber]bool)
					p.Subscribers[topic][sub] = true
					continue
				}

				subs[sub] = true

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

		sub := &Subscriber{
			Channel: make(chan []byte, cap),
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
