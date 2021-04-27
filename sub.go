package pubsub

type PublishedMessage struct {
	Topic string
	Data  []byte
}

type Subscriber struct {
	Id      uint64
	Channel chan *PublishedMessage
	Topics  map[string]bool
}
