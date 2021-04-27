package pubsub

type Message struct {
	Topics []string
	Data   []byte
}

type PubSub struct {
	Message     <-chan Message
	Subscribers map[string][]*Subscriber
}

func New() *PubSub {
	return &PubSub{
		Message:     make(<-chan Message, 256),
		Subscribers: make(map[string][]*Subscriber),
	}
}
