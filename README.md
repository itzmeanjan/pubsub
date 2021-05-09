# pubsub
Embeddable Lightweight Pub/Sub in Go

## Motivation

After using several Pub/Sub systems in writing production grade softwares for sometime, I decided to write one very simple, embeddable, light-weight Pub/Sub system using only native Go functionalities i.e. Go routines, Go channels, Streaming I/O.

Actually, Go channels are **MPSC** i.e. multiple producers can push onto same channel, but there's only one consumer. You're very much free to use multiple consumers on single channel, but they will start competing for messages being published on channel i.e. all consumers won't see all messages published on channel.

Good thing is that Go channels are concurrent-safe. So I considered extending it to make in-application communication more flexible, while leveraging Go's powerful streaming I/O for fast copying of messages from publisher stream to N-consumer stream(s). Below is what provided by this embeddable piece of software.

✌️ | Producer | Consumer
--- | --: | --:
Single | ✅ | ✅
Multiple | ✅ | ✅

## Design

![architecture](./sc/architecture.jpg)

## Stress Testing

For stress testing the system, I wrote one configurable [program](./stress) which makes running tests easy using various CLI argument combination.

Run it using with

```bash
go run stress/main.go -help
go run stress/main.go
```

![stress_testing_result](./sc/result.jpg)

## Usage

First create a Go project with **GOMOD** support.

```bash
# Assuming you're on UNIX flavoured OS

cd
mkdir test_pubsub

cd test_pubsub
go mod init github.com/<username>/test_pubsub

touch main.go
```

Now add `github.com/itzmeanjan/pubsub` as your project dependency

```bash
go get github.com/itzmeanjan/pubsub # v0.1.5 latest
```

And follow full [example](./example/main.go).

---

**Important**

If you're planning to use `pubsub` in your application

- You should first start pub/sub broker using

```go
broker := pubsub.New(ctx)

// concurrent-safe utility method
if !broker.IsAlive() {
	return
}

// Start using broker 👇
```

- If you're a publisher, you should concern yourself with only `PubSub.Publish(...)`

```go
msg := pubsub.Message{
    Topics: []pubsub.String{pubsub.String("topic_1")},
    Data:   []byte("hello"),
}
ok, publishedTo := broker.Publish(&msg) // concurrent-safe
```

- If you're a subscriber, you should first subscribe to `N`-many topics, using `PubSub.Subscribe(...)`. 

```go
subscriber := broker.Subscribe(ctx, 16, []string{"topic_1"}...)
```

- You can start consuming messages using `Subscriber.Next(...)`

```go
for {
    msg := subscriber.Next()
    if msg == nil {
        continue
    }
}
```

- Add more subscriptions on-the-fly using `Subscriber.AddSubscription(...)`

```go
ok, subTo := subscriber.AddSubscription([]string{"topic_2"}...)
```

- Unsubscribe from specific topic using `Subscriber.Unsubscribe(...)`

```go
ok, unsubFrom := subscriber.Unsubscribe([]string{"topic_1"}...)
```

- Unsubscribe from all topics using `Subscriber.UnsubscribeAll(...)`

```go
ok, unsubFrom := subscriber.UnsubscribeAll()
```

- Cancel context when you're done using subscriber **( or may be broker itself, but please be careful ❗️ )**

```go
cancel()
<-time.After(time.Duration(100) * time.Microsecond)
// all good
```

**And all set 🚀**

---

You can check package documentation [here](https://pkg.go.dev/github.com/itzmeanjan/pubsub)
