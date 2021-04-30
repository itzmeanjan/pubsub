package main

//go:generate go run main.go

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
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

func simulate(target uint64, subsC uint64) (bool, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	subscribers := make([]*pubsub.Subscriber, 0, subsC)

	for i := 0; i < int(subsC); i++ {

		subscriber := broker.Subscribe(target, "topic_1")
		if subscriber == nil {
			log.Printf("Failed to subscribe\n")
			return false, 0
		}

		subscribers = append(subscribers, subscriber)

	}

	producerSig := make(chan struct{})

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

		close(producerSig)

	}()

	consumerSig := make(chan time.Duration, subsC)

	for i := 0; i < len(subscribers); i++ {

		go func(subscriber *pubsub.Subscriber) {

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

			<-producerSig
			consumerSig <- time.Since(startedAt)

		}(subscribers[i])

	}

	var received int
	var totalTimeSpent time.Duration

	for timeSpent := range consumerSig {

		totalTimeSpent += timeSpent

		received++
		if received >= len(subscribers) {
			break
		}

	}

	return true, totalTimeSpent / time.Duration(received)

}

func main() {

	f, _ := os.OpenFile("../spmc.csv", os.O_TRUNC|os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	defer func() {

		f.Close()
		log.Printf("Exported statistics in `../spmc.csv`\n")

	}()

	for i := 1; i <= 1024; i *= 2 {

		target := uint64(i * 1024)

		for j := 2; j <= 8; j *= 2 {

			ok, timeTaken := simulate(target, uint64(j))
			if !ok {
				log.Printf("❌ %s\n", datasize.KB*datasize.ByteSize(target))
				continue
			}

			log.Printf("✅ %s with %d subscriber(s) in %s\n", datasize.KB*datasize.ByteSize(target), j, timeTaken)

			f.WriteString(fmt.Sprintf("%s, %d, %d\n", datasize.KB*datasize.ByteSize(target), j, timeTaken))

		}

	}

}
