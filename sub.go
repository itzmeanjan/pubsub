package pubsub

type PublishedMessage struct {
	Topic string
	Data  []byte
}

type Subscriber struct {
	Channel chan *PublishedMessage
	Topics  map[string]bool
}
