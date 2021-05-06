package pubsub

import (
	"bytes"
	"context"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {

	pubsub := New()
	if pubsub.Alive {
		t.Errorf("Expected Pub/Sub system to be dead")
	}

	var (
		TOPIC_1  = "topic_1"
		TOPIC_2  = "topic_2"
		DATA     = []byte("hello")
		TOPICS_1 = []string{TOPIC_1}
		TOPICS_2 = []string{TOPIC_1, TOPIC_2}
		DURATION = time.Duration(1) * time.Millisecond
	)

	msg := Message{Topics: TOPICS_1, Data: DATA}
	published, count := pubsub.Publish(&msg)
	if published {
		t.Errorf("Expected failure in publishing to topic")
	}
	if count != 0 {
		t.Errorf("Expected subscriber count to be 0, got %d", count)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go pubsub.Start(ctx)
	<-time.After(DURATION)
	if !pubsub.Alive {
		t.Errorf("Expected Pub/Sub system to be alive")
	}

	published, count = pubsub.Publish(&msg)
	if !published {
		t.Errorf("Expected to be able to publish to topic")
	}
	if count != 0 {
		t.Errorf("Expected subscriber count to be 0, got %d", count)
	}

	subscriber := pubsub.Subscribe(ctx, 16)
	if subscriber != nil {
		t.Errorf("Expected no creation of subscriber")
	}

	subscriber = pubsub.Subscribe(ctx, 16, TOPIC_1)
	if subscriber == nil && subscriber.id != 1 {
		t.Errorf("Expected creation of subscriber")
	}

	published, count = pubsub.Publish(&msg)
	if !published {
		t.Errorf("Expected to be able to publish to topic")
	}
	if count != 1 {
		t.Errorf("Expected subscriber count to be 1, got %d", count)
	}

	<-time.After(DURATION)

	publishedMessage := subscriber.Next()
	if publishedMessage == nil {
		t.Fatalf("Expected to receive message")
	}
	if !bytes.Equal(publishedMessage.Data, DATA) {
		t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
	}

	msg = Message{Topics: TOPICS_2, Data: DATA}
	published, count = pubsub.Publish(&msg)
	if !published {
		t.Errorf("Expected to be able to publish to topic")
	}
	if count != 1 {
		t.Errorf("Expected subscriber count to be 1, got %d", count)
	}

	state, count := subscriber.AddSubscription()
	if !state {
		t.Errorf("Expected to be able to add new topic subscriptions")
	}
	if count != 0 {
		t.Errorf("Expected to subscribe to 0 new topic, got %d", count)
	}

	state, count = subscriber.AddSubscription(TOPICS_2...)
	if !state {
		t.Errorf("Expected to be able to add new topic subscriptions")
	}
	if count != 1 {
		t.Errorf("Expected to subscribe to 1 new topic, got %d", count)
	}

	<-time.After(DURATION)

	publishedMessage = subscriber.Next()
	if publishedMessage == nil {
		t.Fatalf("Expected to receive message")
	}
	if !bytes.Equal(publishedMessage.Data, DATA) {
		t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
	}

	for i := 0; i < 8; i++ {

		published, count = pubsub.Publish(&msg)
		if !published {
			t.Errorf("Expected to be able to publish to topic")
		}
		if count != 2 {
			t.Errorf("Expected subscriber count to be 2, got %d", count)
		}

	}

	<-time.After(DURATION)

	for i := 0; i < 16; i++ {

		publishedMessage = subscriber.Next()
		if publishedMessage == nil {
			t.Fatalf("Expected to receive message")
		}
		if !bytes.Equal(publishedMessage.Data, DATA) {
			t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
		}

	}

	state, count = subscriber.Unsubscribe()
	if !state {
		t.Errorf("Expected to be able to unsubscribe")
	}
	if count != 0 {
		t.Errorf("Expected to unsubscribe from 0 topic, got %d", count)
	}

	state, count = subscriber.Unsubscribe(TOPIC_1)
	if !state {
		t.Errorf("Expected to be able to unsubscribe")
	}
	if count != 1 {
		t.Errorf("Expected to unsubscribe from 1 topic, got %d", count)
	}

	state, count = subscriber.UnsubscribeAll()
	if !state {
		t.Errorf("Expected to be able to unsubscribe")
	}
	if count != 1 {
		t.Errorf("Expected to unsubscribe from 1 topic, got %d", count)
	}

	state, count = subscriber.Unsubscribe(TOPICS_2...)
	if !state {
		t.Errorf("Expected to be able to unsubscribe")
	}
	if count != 0 {
		t.Errorf("Expected to unsubscribe from 0 topic, got %d", count)
	}

	state, count = subscriber.UnsubscribeAll()
	if !state {
		t.Errorf("Expected to be able to unsubscribe")
	}
	if count != 0 {
		t.Errorf("Expected to unsubscribe from 0 topic, got %d", count)
	}

	cancel()
	<-time.After(DURATION)
	if pubsub.Alive {
		t.Errorf("Expected Pub/Sub system to be dead")
	}

	published, _ = pubsub.Publish(&msg)
	if published {
		t.Errorf("Expected pub/sub system to be down")
	}

	state, _ = subscriber.AddSubscription()
	if state {
		t.Errorf("Expected pub/sub system to be down")
	}

	state, _ = subscriber.Unsubscribe()
	if state {
		t.Errorf("Expected pub/sub system to be down")
	}

	state, _ = subscriber.UnsubscribeAll()
	if state {
		t.Errorf("Expected pub/sub system to be down")
	}

	subscriber = pubsub.Subscribe(context.Background(), 16)
	if subscriber != nil {
		t.Errorf("Expected pub/sub system to be down")
	}

}
