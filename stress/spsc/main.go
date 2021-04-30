package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/itzmeanjan/pubsub"
)

func getRandomByteSlice(len int) []byte {
	buffer := make([]byte, len)

	for i := 0; i < len; i++ {
		buffer[i] = byte(rand.Intn(256))
	}

	return buffer
}

func simulate(target uint64) (bool, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	subscriber := broker.Subscribe(target, "topic_1")
	if subscriber == nil {
		log.Printf("Failed to subscribe\n")
		return false, 0
	}

	signal := make(chan struct{})

	go func() {

		msg := pubsub.Message{
			Topics: []string{"topic_1"},
			Data:   getRandomByteSlice(1024),
		}

		var done, i uint64
		for ; i < target; i++ {

			ok, c := broker.Publish(&msg)
			if !ok {
				break
			}

			done += c

		}

		close(signal)

	}()

	<-time.After(time.Duration(1) * time.Microsecond)

	var startedAt = time.Now()
	var done uint64

	for {

		msg := subscriber.Next()
		if msg == nil {
			continue
		}

		done++
		if done == target {
			break
		}

	}

	<-signal
	return true, time.Since(startedAt)

}

func main() {

	for i := 1; i <= 1024; i *= 2 {

		target := uint64(i * 1024)

		ok, timeTaken := simulate(target)
		if !ok {
			log.Printf("❌ %s\n", datasize.KB*datasize.ByteSize(target))
			continue
		}

		log.Printf("✅ %s in %s\n", datasize.KB*datasize.ByteSize(target), timeTaken)

	}

}
