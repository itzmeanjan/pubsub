package pubsub

import "context"

type Message struct {
	Topics []string
	Data   []byte
}

type PubSub struct {
	Alive         bool
	MessageChan   chan *Message
	SubscribeChan chan *Subscriber
	Subscribers   map[string]map[*Subscriber]bool
}

func New() *PubSub {
	return &PubSub{
		Alive:         true,
		MessageChan:   make(chan *Message, 256),
		SubscribeChan: make(chan *Subscriber, 256),
		Subscribers:   make(map[string]map[*Subscriber]bool),
	}
}

func (p *PubSub) Start(ctx context.Context) {

	for {

		select {

		case <-ctx.Done():
			p.Alive = false
			return

		case m := <-p.MessageChan:

			for i := 0; i < len(m.Topics); i++ {

				subs, ok := p.Subscribers[m.Topics[i]]
				if !ok {
					continue
				}

				for sub := range subs {
					sub.Channel <- m.Data
				}

			}

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

func (p *PubSub) Publish(msg *Message) bool {

	if p.Alive {
		p.MessageChan <- msg
		return true
	}

	return false

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
