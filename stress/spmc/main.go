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
	"github.com/olekukonko/tablewriter"
)

func getRandomByteSlice(len int) []byte {
	buffer := make([]byte, len)

	for i := 0; i < len; i++ {
		buffer[i] = byte(rand.Intn(256))
	}

	return buffer
}

func simulate(target uint64, subsC uint64) (bool, uint64, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	subscribers := make([]*pubsub.Subscriber, 0, subsC)

	for i := 0; i < int(subsC); i++ {
		subscriber := broker.Subscribe(target, "topic_1", "producer_dead")
		if subscriber == nil {
			log.Printf("Failed to subscribe\n")
			return false, 0, 0
		}

		subscribers = append(subscribers, subscriber)
	}

	go func() {
		msg := pubsub.Message{
			Topics: []string{"topic_1"},
			Data:   getRandomByteSlice(1024),
		}

		var i uint64
		for ; i < target; i++ {
			ok, _ := broker.Publish(&msg)
			if !ok {
				break
			}
		}

		broker.Publish(&pubsub.Message{Topics: []string{"producer_dead"}})
	}()

	consumerSig := make(chan struct {
		duration time.Duration
		consumed uint64
	}, subsC)

	for i := 0; i < len(subscribers); i++ {

		go func(subscriber *pubsub.Subscriber) {

			var startedAt = time.Now()
			var consumed uint64
			var producerSignaled bool

			for {
				msg := subscriber.Next()
				if msg == nil {
					if producerSignaled {
						break
					}
					continue
				}

				if msg.Topic == "producer_dead" {
					producerSignaled = true
					continue
				}

				consumed += uint64(len(msg.Data))
			}

			consumerSig <- struct {
				duration time.Duration
				consumed uint64
			}{
				duration: time.Since(startedAt),
				consumed: consumed,
			}

		}(subscribers[i])

	}

	var received int
	var totalTimeSpent time.Duration
	var totalConsumed uint64

	for data := range consumerSig {

		totalTimeSpent += data.duration
		totalConsumed += data.consumed

		received++
		if received >= len(subscribers) {
			break
		}

	}

	return true, totalConsumed, totalTimeSpent / time.Duration(received)

}

func main() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Published Data", "Total Published Data", "Consumer(s)", "Total Consumed Data", "Time"})
	table.SetCaption(true, "Single Producer Multiple Consumers")

	for i := 1; i <= 1024; i *= 2 {

		target := uint64(i * 1024)

		var j uint64 = 2
		for ; j <= 8; j *= 2 {
			ok, consumed, timeTaken := simulate(target, j)
			if !ok {
				continue
			}

			_buffer := []string{
				(datasize.KB * datasize.ByteSize(target)).String(),
				(datasize.KB * datasize.ByteSize(target*j)).String(),
				fmt.Sprintf("%d", j),
				(datasize.B * datasize.ByteSize(consumed)).String(),
				timeTaken.String(),
			}
			table.Append(_buffer)
		}

	}

	table.Render()

}
