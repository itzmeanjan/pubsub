package pubsub

type Subscriber struct {
	Channel <-chan []byte
	Topics  map[string]bool
}
