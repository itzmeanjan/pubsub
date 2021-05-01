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

func simulate(target uint64, prodsC uint64) (bool, uint64, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	subscriber := broker.Subscribe(target*prodsC, "topic_1")
	if subscriber == nil {
		log.Printf("Failed to subscribe\n")
		return false, 0, 0
	}

	signal := make(chan struct{})

	for i := 0; i < int(prodsC); i++ {

		go func() {
			msg := pubsub.Message{
				Topics: []string{"topic_1"},
				Data:   getRandomByteSlice(1024),
			}

			var i uint64
			for ; i < target; i++ {

				if ok, _ := broker.Publish(&msg); !ok {
					break
				}

			}

			signal <- struct{}{}
		}()

	}

	var startedAt = time.Now()
	var sigC, consumed uint64

	for {

		select {
		case <-signal:
			sigC++

		default:
			msg := subscriber.Next()
			if msg == nil {
				if sigC == prodsC {
					return true, consumed, time.Since(startedAt)
				}
				break
			}

			consumed += uint64(len(msg.Data))
		}

	}

}

func main() {

	for i := 1; i <= 1024; i *= 2 {
		target := uint64(i * 1024)

		var j uint64 = 1
		for ; j <= 16; j *= 2 {
			ok, consumed, timeTaken := simulate(target/j, j)
			if !ok {
				log.Printf("❌ %s | %d prod\n", datasize.KB*datasize.ByteSize(target), j)
				continue
			}

			log.Printf("%s ➡️ || ➡️ %s | %d prod | %s\n", datasize.KB*datasize.ByteSize(target), datasize.B*datasize.ByteSize(consumed), j, timeTaken)
		}
	}

}
