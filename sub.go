package pubsub

type PublishedMessage struct {
	Topic string
	Data  []byte
}

type SubscriptionRequest struct {
	Subscriber   *Subscriber
	ResponseChan chan uint64
}

type UnsubscriptionRequest struct {
	Id           uint64
	Topics       []string
	ResponseChan chan uint64
}

type Subscriber struct {
	Id      uint64
	Channel chan *PublishedMessage
	Topics  map[string]bool
}

func (s *Subscriber) AddSubscription(pubsub *PubSub, topics ...string) (bool, uint64) {
	return pubsub.AddSubscription(s, topics...)
}

func (s *Subscriber) Unsubscribe(pubsub *PubSub, topics ...string) (bool, uint64) {
	return pubsub.Unsubscribe(s, topics...)
}

func (s *Subscriber) UnsubcribeAll(pubsub *PubSub) (bool, uint64) {
	return pubsub.UnsubscribeAll(s)
}
