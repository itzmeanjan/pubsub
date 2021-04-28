package pubsub

import (
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
		t.Errorf("Expected initial subscriber index to be 1, got : %d", pubsub.Index)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go pubsub.Start(ctx)
	<-time.After(time.Duration(5) * time.Second)

	if !pubsub.Alive {
		t.Errorf("Expected Pub/Sub system to be alive")
	}

	cancel()
	<-time.After(time.Duration(5) * time.Second)

	if pubsub.Alive {
		t.Errorf("Expected Pub/Sub system to be dead")
	}

}
