package main

//go:generate go run main.go

import (
	"context"
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

func simulate(target uint64) (bool, uint64, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	subscriber := broker.Subscribe(target, "topic_1")
	if subscriber == nil {
		log.Printf("Failed to subscribe\n")
		return false, 0, 0
	}

	signal := make(chan struct{})

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

		signal <- struct{}{}

	}()

	var startedAt = time.Now()
	var consumed uint64
	var signaled bool

	for {

		select {
		case <-signal:
			signaled = true

		default:
			msg := subscriber.Next()
			if msg == nil {
				if signaled {
					return true, consumed, time.Since(startedAt)
				}
				break
			}

			consumed += uint64(len(msg.Data))
		}

	}

}

func main() {

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Produced Data", "Consumed Data", "Time"})
	table.SetCaption(true, "Single Producer Single Consumer")

	for i := 1; i <= 1024; i *= 2 {

		target := uint64(i * 1024)

		ok, consumed, timeTaken := simulate(target)
		if !ok {
			continue
		}

		_buffer := []string{
			(datasize.KB * datasize.ByteSize(target)).String(),
			(datasize.B * datasize.ByteSize(consumed)).String(),
			timeTaken.String(),
		}
		table.Append(_buffer)

	}

	table.Render()

}
