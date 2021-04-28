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
	if pubsub.Index != 1 {
		t.Errorf("Expected subscriber id to be 1, got %d", pubsub.Index)
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

	subscriber := pubsub.Subscribe(16)
	if subscriber != nil {
		t.Errorf("Expected no creation of subscriber")
	}

	subscriber = pubsub.Subscribe(16, TOPIC_1)
	if subscriber.Id != 1 {
		t.Errorf("Expected subscriber id to be 1, got %d", subscriber.Id)
	}

	state, ok := subscriber.Topics[TOPIC_1]
	if !ok || !state {
		t.Errorf("Expected subscriber to be subscribed to `%s`", TOPIC_1)
	}

	publishedMessage := subscriber.BNext(DURATION)
	if publishedMessage != nil {
		t.Fatalf("Expected to receive no message")
	}

	published, count = pubsub.Publish(&msg)
	if !published {
		t.Errorf("Expected to be able to publish to topic")
	}
	if count != 1 {
		t.Errorf("Expected subscriber count to be 1, got %d", count)
	}

	publishedMessage = subscriber.Next()
	if publishedMessage == nil {
		t.Fatalf("Expected to receive message")
	}
	if publishedMessage.Topic != TOPIC_1 {
		t.Errorf("Expected to receive message from `%s`, got from `%s`", TOPIC_1, publishedMessage.Topic)
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

	state, count = subscriber.AddSubscription(pubsub)
	if !state {
		t.Errorf("Expected to be able to add new topic subscriptions")
	}
	if count != 0 {
		t.Errorf("Expected to subscribe to 0 new topic, got %d", count)
	}

	state, count = subscriber.AddSubscription(pubsub, TOPICS_2...)
	if !state {
		t.Errorf("Expected to be able to add new topic subscriptions")
	}
	if count != 1 {
		t.Errorf("Expected to subscribe to 1 new topic, got %d", count)
	}

	state, ok = subscriber.Topics[TOPIC_2]
	if !ok || !state {
		t.Errorf("Expected subscriber to be subscribed to `%s`", TOPIC_2)
	}

	published, count = pubsub.Publish(&msg)
	if !published {
		t.Errorf("Expected to be able to publish to topic")
	}
	if count != 2 {
		t.Errorf("Expected subscriber count to be 2, got %d", count)
	}

	{
		publishedMessage = subscriber.Next()
		if publishedMessage == nil {
			t.Fatalf("Expected to receive message")
		}
		if publishedMessage.Topic != TOPIC_1 {
			t.Errorf("Expected to receive message from `%s`, got from `%s`", TOPIC_1, publishedMessage.Topic)
		}
		if !bytes.Equal(publishedMessage.Data, DATA) {
			t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
		}

		publishedMessage = subscriber.Next()
		if publishedMessage == nil {
			t.Fatalf("Expected to receive message")
		}
		if publishedMessage.Topic != TOPIC_1 {
			t.Errorf("Expected to receive message from `%s`, got from `%s`", TOPIC_1, publishedMessage.Topic)
		}
		if !bytes.Equal(publishedMessage.Data, DATA) {
			t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
		}

		publishedMessage = subscriber.Next()
		if publishedMessage == nil {
			t.Fatalf("Expected to receive message")
		}
		if publishedMessage.Topic != TOPIC_2 {
			t.Errorf("Expected to receive message from `%s`, got from `%s`", TOPIC_2, publishedMessage.Topic)
		}
		if !bytes.Equal(publishedMessage.Data, DATA) {
			t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
		}
	}

	state, count = subscriber.Unsubscribe(pubsub)
	if !state {
		t.Errorf("Expected to be able to unsubscribe")
	}
	if count != 0 {
		t.Errorf("Expected to unsubscribe from 0 topic, got %d", count)
	}

	state, count = subscriber.Unsubscribe(pubsub, TOPIC_1)
	if !state {
		t.Errorf("Expected to be able to unsubscribe")
	}
	if count != 1 {
		t.Errorf("Expected to unsubscribe from 1 topic, got %d", count)
	}

	state = subscriber.Topics[TOPIC_1]
	if state {
		t.Errorf("Expected to unsubscribe from `%s`", TOPIC_1)
	}

	state, count = subscriber.UnsubscribeAll(pubsub)
	if !state {
		t.Errorf("Expected to be able to unsubscribe")
	}
	if count != 1 {
		t.Errorf("Expected to unsubscribe from 1 topic, got %d", count)
	}

	state = subscriber.Topics[TOPIC_2]
	if state {
		t.Errorf("Expected to unsubscribe from `%s`", TOPIC_2)
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

	state, _ = subscriber.AddSubscription(pubsub)
	if state {
		t.Errorf("Expected pub/sub system to be down")
	}

	state, _ = subscriber.Unsubscribe(pubsub)
	if state {
		t.Errorf("Expected pub/sub system to be down")
	}

	state, _ = subscriber.UnsubscribeAll(pubsub)
	if state {
		t.Errorf("Expected pub/sub system to be down")
	}

	subscriber = pubsub.Subscribe(16)
	if subscriber != nil {
		t.Errorf("Expected pub/sub system to be down")
	}

}
