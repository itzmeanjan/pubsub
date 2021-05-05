package main

import (
	"context"
	"flag"
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

func simulate(target uint64, prodsC uint64, unsafe bool) (bool, uint64, time.Duration) {

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

	var unsafe = flag.Bool("unsafe", false, "avoid copying messages for each subcriber i.e. FASTer")
	flag.Parse()

	table := tablewriter.NewWriter(os.Stdout)
	table.SetHeader([]string{"Data", "Producer(s)", "Total Produced", "Consumed Data", "Time"})
	table.SetCaption(true, "Multiple Producers Single Consumer")

	for i := 1; i <= 1024; i *= 2 {
		target := uint64(i * 1024)

		var j uint64 = 2
		for ; j <= 8; j *= 2 {
			ok, consumed, timeTaken := simulate(target/j, j, *unsafe)
			if !ok {
				continue
			}

			_buffer := []string{
				(datasize.KB * datasize.ByteSize(target/j)).String(),
				fmt.Sprintf("%d", j),
				(datasize.KB * datasize.ByteSize(target)).String(),
				(datasize.B * datasize.ByteSize(consumed)).String(),
				timeTaken.String(),
			}
			table.Append(_buffer)
		}
	}

	table.Render()

}
