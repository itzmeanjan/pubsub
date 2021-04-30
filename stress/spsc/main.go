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

func simulate(target uint64, cap uint64) (bool, time.Duration, time.Duration) {

	broker := pubsub.New()

	ctx, cancel := context.WithCancel(context.Background())
	go broker.Start(ctx)
	defer cancel()

	<-time.After(time.Duration(100) * time.Microsecond)

	subscriber := broker.Subscribe(cap, "topic_1")
	if subscriber == nil {
		log.Printf("Failed to subscribe\n")
		return false, 0, 0
	}

	var publisherTime, consumerTime time.Duration
	signal := make(chan struct{})

	go func() {

		startedAt := time.Now()
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

		publisherTime = time.Since(startedAt)
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

	consumerTime = time.Since(startedAt)
	<-signal

	return true, publisherTime, consumerTime

}

func drawChart(xAxis []string, producer []opts.BarData, consumer []opts.BarData, sink string) {

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
		AddSeries("Producer", producer).
		AddSeries("Consumer", consumer).
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
	var producer = make([]opts.BarData, 0)
	var consumer = make([]opts.BarData, 0)

	for i := 1; i <= 1024; i *= 2 {

		target := uint64(i * 1024)

		ok, publisherTime, consumerTime := simulate(target, 1024)
		if !ok {
			log.Printf("❌ %s\n", datasize.KB*datasize.ByteSize(target))
			continue
		}

		log.Printf("✅ %s :: Producer : %s, Consumer : %s\n", datasize.KB*datasize.ByteSize(target), publisherTime, consumerTime)

		xAxis = append(xAxis, (datasize.KB * datasize.ByteSize(target)).String())
		producer = append(producer, opts.BarData{
			Value: publisherTime / (time.Duration(1) * time.Millisecond),
		})
		consumer = append(consumer, opts.BarData{
			Value: consumerTime / (time.Duration(1) * time.Millisecond),
		})

	}

	drawChart(xAxis, producer, consumer, "../charts/bar.html")

}
