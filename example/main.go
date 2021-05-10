package main

import (
	"context"
	"log"
	"time"

	"github.com/itzmeanjan/pubsub"
)

func main() {
	// -- Very important, starting pub/sub system
	ctx, cancel := context.WithCancel(context.Background())
	broker := pubsub.New(ctx)
	defer cancel()
	// -- Starting pub/sub system

	if !broker.IsAlive() {
		log.Printf("❌ Failed to start pub/sub hub\n")
		return
	}

	subscriber := broker.Subscribe(ctx, 16, "topic_1", "topic_2")
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
		Topics: []pubsub.String{
			pubsub.String("topic_1"),
			pubsub.String("topic_2"),
			pubsub.String("topic_3"),
		},
		Data: []byte("hello"),
	}
	published, on := broker.Publish(&msg)
	if !published {
		log.Printf("Failed to publish message to topics\n")
		return
	}

	log.Printf("✅ Published `hello` to %d topics\n", on)

	var receivedC uint64
	for range subscriber.Listener {
		msg := subscriber.Next()
		if msg == nil {
			break
		}

		log.Printf("✅ Received `%s` on topic `%s`\n", msg.Data, msg.Topic)

		receivedC++
		if receivedC >= 2 {
			break
		}
	}

	// Subscribe to new topic using same subscriber instance
	if subscribed, _ := subscriber.AddSubscription("topic_3"); subscribed {
		log.Printf("✅ Subscribed to `topic_3`\n")
	}

	if unsubscribed, _ := subscriber.Unsubscribe("topic_1"); unsubscribed {
		log.Printf("✅ Unsubscribed from `topic_1`\n")
	}

	if unsubscribed, from := subscriber.UnsubscribeAll(); unsubscribed {
		log.Printf("✅ Unsubscribed from %d topic(s)\n", from)
	}

	cancel()
	<-time.After(time.Duration(100) * time.Microsecond)

	if broker.IsAlive() {
		log.Printf("❌ Failed to stop pub/sub hub\n")
	}

}
