package main

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

func simulate(target uint64, parties uint64) (bool, uint64, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	subscribers := make([]*pubsub.Subscriber, 0, parties)
	for i := 0; i < int(parties); i++ {

		subscriber := broker.Subscribe(target*parties, "topic_1", "producer_dead")
		if subscriber == nil {
			log.Printf("Failed to subscribe\n")
			return false, 0, 0
		}
		subscribers = append(subscribers, subscriber)

	}

	for i := 0; i < int(parties); i++ {

		go func(i int) {
			msg := pubsub.Message{
				Topics: []string{"topic_1"},
				Data:   getRandomByteSlice(1024),
			}

			var j, total uint64
			for ; j < target; j++ {
				if ok, _ := broker.Publish(&msg); !ok {
					break
				}

				total += uint64(len(msg.Data))
			}

			broker.Publish(&pubsub.Message{Topics: []string{"producer_dead"}})
		}(i)

	}

	consumerSig := make(chan struct {
		duration time.Duration
		consumed uint64
	}, parties)

	for i := 0; i < len(subscribers); i++ {

		go func(subscriber *pubsub.Subscriber) {

			var startedAt = time.Now()
			var consumed uint64
			var signaledBy uint64

			for {
				msg := subscriber.Next()
				if msg == nil {
					if signaledBy == parties {
						break
					}
					continue
				}

				if msg.Topic == "producer_dead" {
					signaledBy++
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
	table.SetHeader([]string{"Data", "Producer(s)", "Total Produced", "Consumer(s)", "Total Consumed", "Time"})
	table.SetCaption(true, "Multiple Producers Multiple Consumers")

	for i := 1; i <= 1024; i *= 2 {
		target := uint64(i * 1024)

		var j uint64 = 2
		for ; j <= 4; j *= 2 {
			ok, consumed, timeTaken := simulate(target/j, j)
			if !ok {
				continue
			}

			_buffer := []string{
				(datasize.KB * datasize.ByteSize(target/j)).String(),
				fmt.Sprintf("%d", j),
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
