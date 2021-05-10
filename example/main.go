package main

import (
	"log"

	"github.com/itzmeanjan/pubsub"
)

func main() {
	broker := pubsub.New(1)

	subscriber := broker.Subscribe(16, "topic_1", "topic_2")
	if subscriber == nil {
		log.Printf("❌ Failed to subscribe to topics\n")
		return
	}

	// Publish arbitrary byte data on N-many topics, without concerning whether all
	// topics having at least 1 subscriber or not
	//
	// During publishing if some topic doesn't have certain subscriber, it won't receive
	// message later when it joins
	msg := pubsub.Message{
		Topics: []string{
			"topic_1",
			"topic_2",
			"topic_3",
		},
		Data: []byte("hello"),
	}

	log.Printf("✅ Published `hello` to %d topics\n", broker.Publish(&msg))

	for range subscriber.Listener() {
		msg := subscriber.Next()
		if msg == nil {
			break
		}

		log.Printf("✅ Received `%s` on topic `%s`\n", msg.Data, msg.Topic)

		if !subscriber.Consumable() {
			break
		}
	}

	if !subscriber.Consumable() {
		log.Printf("✅ Consumed all buffered messages\n")
	}

	// Subscribe to new topic using same subscriber instance
	if subscriber.AddSubscription("topic_3") == 1 {
		log.Printf("✅ Subscribed to `topic_3`\n")
	}

	if subscriber.Unsubscribe("topic_1") == 1 {
		log.Printf("✅ Unsubscribed from `topic_1`\n")
	}

	log.Printf("✅ Published `hello` to %d topics\n", broker.Publish(&msg))

	for range subscriber.Listener() {
		msg := subscriber.Next()
		if msg == nil {
			break
		}

		log.Printf("✅ Received `%s` on topic `%s`\n", msg.Data, msg.Topic)

		if !subscriber.Consumable() {
			break
		}
	}

	if !subscriber.Consumable() {
		log.Printf("✅ Consumed all buffered messages\n")
	}

	log.Printf("✅ Unsubscribed from %d topic(s)\n", subscriber.UnsubscribeAll())

	subscriber.Destroy()
	log.Printf("✅ Destroyed subscriber\n")

}
