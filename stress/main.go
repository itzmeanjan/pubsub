package main

import (
	"context"
	crand "crypto/rand"
	"flag"
	"fmt"
	"log"
	mrand "math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/gookit/color"
	"github.com/itzmeanjan/pubsub"
)

func getRandomByteSlice(len int) []byte {
	buffer := make([]byte, len)

	n, err := crand.Read(buffer)
	if err != nil && n == len {
		return buffer
	}

	for i := 0; i < len; i++ {
		buffer[i] = byte(mrand.Intn(256))
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

func simulate(ctx context.Context, producers int, consumers int, topics int, rollAfter time.Duration, chunkSize datasize.ByteSize) {

	broker := pubsub.New(uint64(producers))
	_topics := generateTopics(topics)

	<-time.After(time.Duration(100) * time.Microsecond)

	subscribers := make([]*pubsub.Subscriber, 0, consumers)
	for i := 0; i < consumers; i++ {

		subscriber := broker.Subscribe(256, _topics...)
		if subscriber == nil {
			return
		}
		subscribers = append(subscribers, subscriber)

	}

	for i := 0; i < producers; i++ {
		go func(i int) {
			var published uint64
			var startedAt = time.Now()

			msg := pubsub.Message{
				Topics: _topics,
				Data:   getRandomByteSlice(int(chunkSize)),
			}

		LOOP:
			for {

				select {
				case <-ctx.Done():
					log.Printf("Shutting down P%d", i)
					break LOOP

				default:
					n := broker.Publish(&msg)
					published += uint64(len(msg.Data)) * n

					if time.Since(startedAt) >= rollAfter {
						log.Println(color.Blue.Sprintf("[P%d: ] at %s/s", i, (datasize.B * datasize.ByteSize(published/uint64(rollAfter/time.Second))).HR()))

						published = 0
						startedAt = time.Now()
					}

				}

			}
		}(i)
	}

	for i := 0; i < len(subscribers); i++ {
		go func(i int, subscriber *pubsub.Subscriber) {
			var consumed uint64
			var startedAt = time.Now()

		LOOP:
			for {
				select {
				case <-ctx.Done():
					log.Printf("Shutting down C%d", i)
					break LOOP

				case <-subscriber.Listener():
					msg := subscriber.Next()
					if msg == nil {
						continue
					}

					consumed += uint64(len(msg.Data))

					if time.Since(startedAt) >= rollAfter {
						log.Println(color.Green.Sprintf("[C%d: ] at %s/s", i, (datasize.B * datasize.ByteSize(consumed/uint64(rollAfter/time.Second))).HR()))

						consumed = 0
						startedAt = time.Now()
					}

				}
			}
		}(i, subscribers[i])
	}

}

func main() {

	var rollAfter = flag.Duration("rollAfter", time.Duration(4)*time.Second, "calculate performance & roll to zero, after duration")
	var producer = flag.Uint64("producer", 2, "#-of producers involved in simulation")
	var consumer = flag.Uint64("consumer", 2, "#-of consumers involved in simulation")
	var topic = flag.Uint64("topic", 2, "#-of topics involved in simulation")
	var chunk = flag.String("chunk", "1 MB", "published message size")
	flag.Parse()

	var _chunk datasize.ByteSize
	if err := _chunk.UnmarshalText([]byte(*chunk)); err != nil {
		log.Printf("Error : %s\n", err.Error())
		_chunk = datasize.MB
	}

	if *rollAfter < time.Second {
		log.Printf("Making rollAfter duration %s\n", time.Second)
		*rollAfter = time.Second
	}

	interruptChan := make(chan os.Signal, 1)
	signal.Notify(interruptChan, syscall.SIGTERM, syscall.SIGINT)

	log.Printf("Pub/Sub Simulation with %d producers, %d consumers, %d topics & %s chunk size\n", *producer, *consumer, *topic, _chunk.HR())
	ctx, cancel := context.WithCancel(context.Background())
	simulate(ctx, int(*producer), int(*consumer), int(*topic), *rollAfter, _chunk)

	<-interruptChan
	cancel()
	<-time.After(time.Second)
	log.Println("Graceful shutdown !")

}
