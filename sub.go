package pubsub

type PublishedMessage struct {
	Topic string
	Data  []byte
}

type UnsubscriptionRequest struct {
	Id     uint64
	Topics []string
}

type Subscriber struct {
	Id      uint64
	Channel chan *PublishedMessage
	Topics  map[string]bool
}
