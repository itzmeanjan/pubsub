package main

import (
	"context"
	"flag"
	"fmt"
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

func generateTopics(count int) []string {
	topics := make([]string, count)

	for i := 0; i < count; i++ {
		topics[i] = fmt.Sprintf("topic_%d", i)
	}

	return topics
}

func simulate(rollAfter time.Duration, parties uint64) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	topics := generateTopics(int(parties))

	subscribers := make([]*pubsub.Subscriber, 0, parties)
	for i := 0; i < int(parties); i++ {

		subscriber := broker.Subscribe(1024, topics...)
		if subscriber == nil {
			return
		}
		subscribers = append(subscribers, subscriber)

	}

	for i := 0; i < int(parties); i++ {
		go func(i int) {

			var published uint64
			var startedAt = time.Now()
			msg := pubsub.Message{
				Topics: topics,
				Data:   getRandomByteSlice(1024),
			}

			for {
				if ok, _ := broker.Publish(&msg); !ok {
					break
				}

				published += uint64(len(msg.Data))

				if time.Since(startedAt) >= rollAfter {
					log.Printf("[P%d: ] at %s/s", i, (datasize.B * datasize.ByteSize(published/uint64(rollAfter/time.Second))).HR())

					published = 0
					startedAt = time.Now()
				}
			}

		}(i)
	}

	for i := 0; i < len(subscribers); i++ {
		go func(i int, subscriber *pubsub.Subscriber) {

			var consumed uint64
			var startedAt = time.Now()

			for {
				msg := subscriber.Next()
				if msg == nil {
					continue
				}

				consumed += uint64(len(msg.Data))

				if time.Since(startedAt) >= rollAfter {
					log.Printf("[C%d: ] at %s/s", i, (datasize.B * datasize.ByteSize(consumed/uint64(rollAfter/time.Second))).HR())

					consumed = 0
					startedAt = time.Now()
				}
			}

		}(i, subscribers[i])
	}

	<-ctx.Done()

}

func main() {

	var rollAfter = flag.Duration("rollAfter", time.Duration(4)*time.Second, "calculate performance & roll to zero, after duration")
	var parties = flag.Uint64("parties", 2, "#-of producers, consumers & topics involved in simulation")
	flag.Parse()

	simulate(*rollAfter, *parties)

}
