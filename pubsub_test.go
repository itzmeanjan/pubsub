package pubsub

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"
)

func TestPubSub(t *testing.T) {

	var (
		TOPIC_1  = "topic_1"
		TOPIC_2  = "topic_2"
		DATA     = []byte("hello")
		TOPICS_1 = []string{TOPIC_1}
		TOPICS_2 = []string{TOPIC_1, TOPIC_2}
		DURATION = time.Duration(1) * time.Millisecond
		msg      = Message{Topics: TOPICS_1, Data: DATA}
	)

	ctx, cancel := context.WithCancel(context.Background())
	pubsub := New(ctx)

	if _, count := pubsub.Publish(&msg); count != 0 {
		t.Errorf("Expected subscriber count to be 0, got %d", count)
	}

	if subscriber := pubsub.Subscribe(ctx, 16); subscriber != nil {
		t.Errorf("Expected no creation of subscriber")
	}

	subscriber := pubsub.Subscribe(ctx, 16, TOPIC_1)
	if subscriber == nil && subscriber.id != 1 {
		t.Errorf("Expected creation of subscriber")
	}

	if subscriber.Consumable() {
		t.Errorf("Expected zero consumable message")
	}

	if _, count := pubsub.Publish(&msg); count != 1 {
		t.Errorf("Expected subscriber count to be 1, got %d", count)
	}

	<-time.After(DURATION)

	if publishedMessage := subscriber.Next(); publishedMessage == nil || !bytes.Equal(publishedMessage.Data, DATA) {
		t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
	}

	msg = Message{Topics: TOPICS_2, Data: DATA}

	if _, count := pubsub.Publish(&msg); count != 1 {
		t.Errorf("Expected subscriber count to be 1, got %d", count)
	}

	if subscriber.AddSubscription() != 0 {
		t.Errorf("Expected to subscribe to 0 topics")
	}
	if c := subscriber.AddSubscription(TOPICS_2[0], TOPICS_2[1]); c != 2 {
		t.Errorf("Expected to subscribe to 2 topics, did %d\n", c)
	}

	<-time.After(DURATION)

	if publishedMessage := subscriber.Next(); publishedMessage == nil || !bytes.Equal(publishedMessage.Data, DATA) {
		t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
	}

	for i := 0; i < 8; i++ {

		if _, count := pubsub.Publish(&msg); count != 2 {
			t.Errorf("Expected subscriber count to be 2, got %d", count)
		}

	}

	<-time.After(DURATION)

	for i := 0; i < 8; i++ {

		if publishedMessage := subscriber.Next(); publishedMessage == nil || strings.Compare(publishedMessage.Topic, TOPIC_1) != 0 || !bytes.Equal(publishedMessage.Data, DATA) {
			t.Errorf("Expected to receive `%s` from `%s`, got `%s` from `%s`", DATA, TOPIC_1, publishedMessage.Data, publishedMessage.Topic)
		}

		if publishedMessage := subscriber.Next(); publishedMessage == nil || strings.Compare(publishedMessage.Topic, TOPIC_2) != 0 || !bytes.Equal(publishedMessage.Data, DATA) {
			t.Errorf("Expected to receive `%s` from `%s`, got `%s` from `%s`", DATA, TOPIC_2, publishedMessage.Data, publishedMessage.Topic)
		}

	}

	if _, count := subscriber.Unsubscribe(); count != 0 {
		t.Errorf("Expected to unsubscribe from 0 topic, got %d", count)
	}

	if _, count := subscriber.Unsubscribe(TOPIC_1); count != 1 {
		t.Errorf("Expected to unsubscribe from 1 topic, got %d", count)
	}

	if _, count := subscriber.UnsubscribeAll(); count != 1 {
		t.Errorf("Expected to unsubscribe from 1 topic, got %d", count)
	}

	if _, count := subscriber.Unsubscribe(TOPICS_2[0], TOPICS_2[1]); count != 0 {
		t.Errorf("Expected to unsubscribe from 0 topic, got %d", count)
	}

	if _, count := subscriber.UnsubscribeAll(); count != 0 {
		t.Errorf("Expected to unsubscribe from 0 topic, got %d", count)
	}

	if !subscriber.Destroy() {
		t.Errorf("Expected to destroy subscriber")
	}

	cancel()
	<-time.After(DURATION)

	if published, _ := pubsub.Publish(&msg); published {
		t.Errorf("Expected pub/sub system to be down")
	}

	if state, _ := subscriber.Unsubscribe(); state {
		t.Errorf("Expected pub/sub system to be down")
	}

	if state, _ := subscriber.UnsubscribeAll(); state {
		t.Errorf("Expected pub/sub system to be down")
	}

	if subscriber = pubsub.Subscribe(context.Background(), 16); subscriber != nil {
		t.Errorf("Expected pub/sub system to be down")
	}

}
