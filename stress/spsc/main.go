package main

//go:generate go run main.go

import (
	"context"
	"flag"
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

func simulate(target uint64, unsafe bool) (bool, uint64, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

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

	var unsafe = flag.Bool("unsafe", false, "avoid copying messages for each subcriber i.e. FASTer")
	flag.Parse()

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Produced Data", "Consumed Data", "Time"})
	table.SetCaption(true, "Single Producer Single Consumer")

	for i := 1; i <= 1024; i *= 2 {

		target := uint64(i * 1024)

		ok, consumed, timeTaken := simulate(target, *unsafe)
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
