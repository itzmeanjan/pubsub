# pubsub
Embeddable Lightweight Pub/Sub in Go

## Motivation

After using several Pub/Sub systems in writing production grade softwares for sometime, I decided to write one very simple, embeddable, light-weight Pub/Sub system using only native Go functionalities i.e. Go routines, Go channels.

Actually, Go channels are **MPSC** i.e. multiple producers can push onto same channel, but there's only one consumer. You're very much free to use multiple consumers on single channel, but they will start competing for messages being published on channel i.e. all consumers won't see all messages published on channel.

Good thing is that Go channels are concurrent-safe. So I considered extending it to make in-application communication more flexible. Below is what provided by this embeddable piece of software.

✌️ | Producer | Consumer
--- | --: | --:
Single | ✅ | ✅
Multiple | ✅ | ✅

## Design

![architecture](./sc/architecture.jpg)

## Stress Testing

Stress testing using `pubsub` was done for following message passing patterns, where every message was of size **1024 bytes** & I attempted to calculate time taken for producer(s) & consumer(s) to respectively publish `{ 1MB, 2MB, 4MB ... 1GB }` data & consume same.

![spsc](./sc/spsc.png)

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
go get github.com/itzmeanjan/pubsub # v0.1.1 latest
```

And follow full [example](./example/main.go).

---

**Important**

A few things you should care about when using `pubsub` in your application

- Though Go channels are heavily used in this package, you're never supposed to be interacting with them directly. Abstracted, easy to use methods are made available for you. **DON'T INTERACT WITH CHANNELS DIRECTLY**
- When creating new subscriber, it'll be allocated with one unique **ID**. ID generation logic is very simple. **BUT MAKE SURE YOU NEVER MANIPULATE IT**

```js
last_id = 1 // initial value
next_id = last_id + 1
last_id = next_id
```
- If you're a publisher, you should concern yourself with only `PubSub.Publish(...)` method.
- If you're a subscriber, you should first subscribe to `N`-many topics, using `PubSub.Subscribe(...)`. You can start reading using
    - `Subscriber.Next()` [ **non-blocking** ]
    - `Subscriber.BNext(...)` [ **blocking** ]
    - `Subscriber.AddSubscription(...)` [ **add more subscriptions on-the-fly** ]
    - `Subscriber.Unsubscribe(...)` [ **cancel some topic subscriptions** ]
    - `Subscriber.UnsubscribeAll(...)` [ **cancel all topic subscriptions** ]
    - `Subscriber.Close()` [ **when you don't want to use this subscriber anymore** ]

- You're good to invoke 👆 methods from `N`-many go routines on same **Pub/Sub System** which you started using `PubSub.Start(...)`. It needs to be first created

```go
broker := PubSub.New()
```

**And all set 🚀**

---

You can check package documentation [here](https://pkg.go.dev/github.com/itzmeanjan/pubsub)
