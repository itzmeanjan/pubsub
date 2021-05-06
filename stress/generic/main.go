package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
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

func simulate(ctx context.Context, rollAfter time.Duration, parties uint64, unsafe bool) {

	broker := pubsub.New()
	go broker.Start(ctx)
	topics := generateTopics(int(parties))

	<-time.After(time.Duration(100) * time.Microsecond)

	// This will make things FASTer, but might
	// not be always a good idea
	//
	// Atleast if you're going to modify same slice
	// which you used for sending one message
	// it'll be reflected at every subscriber's side
	//
	// Caution : Use with care
	if unsafe {
		broker.AllowUnsafe()
	}

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

}

func main() {

	var rollAfter = flag.Duration("rollAfter", time.Duration(4)*time.Second, "calculate performance & roll to zero, after duration")
	var parties = flag.Uint64("parties", 2, "#-of producers, consumers & topics involved in simulation")
	var unsafe = flag.Bool("unsafe", false, "avoid copying messages for each subcriber i.e. FASTer")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, syscall.SIGTERM, syscall.SIGINT)

	log.Printf("Pub/Sub Simulation with %d Producers, Consumers & Topics\n", *parties)
	simulate(ctx, *rollAfter, *parties, *unsafe)

	<-interruptChan
	cancel()

	<-time.After(time.Duration(1) * time.Second)
	log.Printf("Graceful shutdown\n")

}
