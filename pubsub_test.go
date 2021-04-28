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
		TOPIC_1 = "topic_1"
		DATA    = []byte("hello")
		TOPICS  = []string{TOPIC_1}
	)

	msg := Message{Topics: TOPICS, Data: DATA}
	published, count := pubsub.Publish(&msg)
	if published {
		t.Errorf("Expected failure in publishing to topic")
	}
	if count != 0 {
		t.Errorf("Expected subscriber count to be 0, got %d", count)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go pubsub.Start(ctx)
	<-time.After(time.Duration(100) * time.Millisecond)
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

	subscriber := pubsub.Subscribe(16, TOPIC_1)
	if subscriber.Id != 1 {
		t.Errorf("Expected subscriber id to be 1, got %d", subscriber.Id)
	}

	state, ok := subscriber.Topics[TOPIC_1]
	if !ok || !state {
		t.Errorf("Expected subscriber to be subscribed to `%s`", TOPIC_1)
	}

	published, count = pubsub.Publish(&msg)
	if !published {
		t.Errorf("Expected to be able to publish to topic")
	}
	if count != 1 {
		t.Errorf("Expected subscriber count to be 1, got %d", count)
	}

	publishedMessage := subscriber.Next()
	if publishedMessage == nil {
		t.Fatalf("Expected to receive message")
	}
	if publishedMessage.Topic != TOPIC_1 {
		t.Errorf("Expected to receive message from `%s`, got from `%s`", TOPIC_1, publishedMessage.Topic)
	}
	if !bytes.Equal(publishedMessage.Data, DATA) {
		t.Errorf("Expected to receive `%s`, got `%s`", DATA, publishedMessage.Data)
	}

	cancel()
	<-time.After(time.Duration(100) * time.Millisecond)
	if pubsub.Alive {
		t.Errorf("Expected Pub/Sub system to be dead")
	}

}
