package pubsub

type PublishedMessage struct {
	Topic string
	Data  Binary
}
