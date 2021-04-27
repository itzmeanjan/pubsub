package pubsub

type Message struct {
	Topics []string
	Data   []byte
}

type PubSub struct {
	Message     <-chan Message
	Subscribers map[string][]*Subscriber
}
