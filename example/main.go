package main

import (
	"context"
	"log"
	"time"

	"github.com/itzmeanjan/pubsub"
)

func main() {
	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()
	// Just waiting little while to give pub/sub broker enough time to get up & running
	<-time.After(time.Duration(1) * time.Millisecond)

	// At max 16 messages to be kept bufferred at a time
	subscriber := broker.Subscribe(16, "topic_1", "topic_2")
	if subscriber == nil {
		log.Printf("Failed to subscribe to topics\n")
		return
	}

	// Publish arbitrary byte data on N-many topics, without concerning whether all
	// topics having at least 1 subscriber or not
	//
	// During publishing if some topic doesn't have certain subscriber, it won't receive
	// message later when it joins
	msg := pubsub.Message{Topics: []string{"topic_1", "topic_2", "topic_3"}, Data: []byte("hello")}
	published, on := broker.Publish(&msg)
	if !published {
		log.Printf("Failed to publish message to topics\n")
		return
	}

	log.Printf("✅ Published `hello` to %d topics\n", on)

	for {
		// Non-blocking calls, returns immediately if finds nothing in buffer
		msg := subscriber.Next()
		if msg == nil {
			break
		}

		log.Printf("✅ Received `%s` on topic `%s`\n", msg.Data, msg.Topic)
	}

	// Attempt to receive from channel if something already available
	//
	// If not found, wait for `duration` & reattempt
	//
	// This time return without blocking
	if msg := subscriber.BNext(time.Duration(1) * time.Millisecond); msg != nil {
		log.Printf("✅ Received `%s` on topic `%s`\n", msg.Data, msg.Topic)
	} else {
		log.Printf("✅ Nothing else to receive\n")
	}

	if subscribed, _ := subscriber.AddSubscription(broker, "topic_3"); subscribed {
		log.Printf("✅ Subscribed to `topic_3`\n")
	}

	if unsubscribed, _ := subscriber.Unsubscribe(broker, "topic_1"); unsubscribed {
		log.Printf("✅ Unsubscribed from `topic_1`\n")
	}

	if unsubscribed, from := subscriber.UnsubscribeAll(broker); unsubscribed {
		log.Printf("✅ Unsubscribed from %d topic(s)\n", from)
	}

	// Calling this multiple times doesn't cause harm, but not required
	if subscriber.Close() {
		log.Printf("✅ Destroyed subscriber\n")
	}

}
