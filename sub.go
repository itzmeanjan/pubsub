package pubsub

type PublishedMessage struct {
	Topic string
	Data  []byte
}

type SubscriptionRequest struct {
	Subscriber   *Subscriber
	ResponseChan chan uint64
}

type Subscriber struct {
	Id      uint64
	Channel chan *PublishedMessage
	Topics  map[string]bool
}

type UnsubscriptionRequest struct {
	Id           uint64
	Topics       []string
	ResponseChan chan uint64
}
