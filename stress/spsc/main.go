package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/c2h5oh/datasize"
	"github.com/go-echarts/go-echarts/v2/charts"
	"github.com/go-echarts/go-echarts/v2/opts"
	"github.com/itzmeanjan/pubsub"
)

func getRandomByteSlice(len int) []byte {
	buffer := make([]byte, len)

	for i := 0; i < len; i++ {
		buffer[i] = byte(rand.Intn(256))
	}

	return buffer
}

func simulate(target uint64, cap uint64) (bool, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	subscriber := broker.Subscribe(cap, "topic_1")
	if subscriber == nil {
		log.Printf("Failed to subscribe\n")
		return false, 0
	}

	signal := make(chan struct{})

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

		close(signal)

	}()

	<-time.After(time.Duration(1) * time.Microsecond)

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

	<-signal
	return true, time.Since(startedAt)

}

func drawChart(xAxis []string, durations []opts.BarData, sink string) {

	bar := charts.NewBar()

	bar.SetGlobalOptions(
		charts.WithTitleOpts(opts.Title{
			Title: "Single Producer Single Consumer",
		}),
		charts.WithLegendOpts(opts.Legend{Right: "80%"}),
		charts.WithXAxisOpts(opts.XAxis{
			Name: "Data",
		}),
		charts.WithYAxisOpts(opts.YAxis{
			Name:      "Time",
			AxisLabel: &opts.AxisLabel{Show: true, Formatter: "{value} ms"},
		}))

	bar.SetXAxis(xAxis).
		AddSeries("Duration", durations).
		SetSeriesOptions(charts.WithLabelOpts(opts.Label{
			Show:      true,
			Position:  "top",
			Formatter: "{c}",
		}))

	f, _ := os.Create(sink)
	bar.Render(f)

}

func main() {

	var xAxis = make([]string, 0)
	var durations = make([]opts.BarData, 0)

	for i := 1; i <= 1024; i *= 2 {

		target := uint64(i * 1024)

		ok, timeTaken := simulate(target, 1024)
		if !ok {
			log.Printf("❌ %s\n", datasize.KB*datasize.ByteSize(target))
			continue
		}

		log.Printf("✅ %s in %s\n", datasize.KB*datasize.ByteSize(target), timeTaken)

		xAxis = append(xAxis, (datasize.KB * datasize.ByteSize(target)).String())
		durations = append(durations, opts.BarData{
			Value: timeTaken / (time.Duration(1) * time.Millisecond),
		})

	}

	drawChart(xAxis, durations, "../charts/spsc.html")
	log.Printf("Plotted bar chart for SPSC\n")

}
