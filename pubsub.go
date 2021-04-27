package pubsub

import "context"

type Message struct {
	Topics []string
	Data   []byte
}

type PubSub struct {
	Alive       bool
	Message     chan *Message
	Subscribers map[string][]*Subscriber
}

func New() *PubSub {
	return &PubSub{
		Alive:       true,
		Message:     make(chan *Message, 256),
		Subscribers: make(map[string][]*Subscriber),
	}
}

func (p *PubSub) Start(ctx context.Context) {

	for {

		select {

		case <-ctx.Done():
			p.Alive = false
			return

		case m := <-p.Message:

			for i := 0; i < len(m.Topics); i++ {

				subs, ok := p.Subscribers[m.Topics[i]]
				if !ok {
					continue
				}

				for j := 0; j < len(subs); j++ {
					subs[j].Channel <- m.Data
				}

			}

		}

	}

}

func (p *PubSub) Publish(msg *Message) bool {
	if p.Alive {
		p.Message <- msg
		return true
	}

	return false
}
