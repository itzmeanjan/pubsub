# pubsub
Embeddable Lightweight Pub/Sub in Go

## Motivation

After using several Pub/Sub systems in writing production softwares for sometime, I decided to write one very simple, embeddable, light-weight Pub/Sub system using only native Go functionalities i.e. Go routines, Go channels.

I found Go channels are **MPSC** i.e. multiple producers can push onto same channel, but there's only one consumer. You're very much free to use multiple consumers on single channel, but they will start competing for messages being published on channel.

Good thing is that Go channels are concurrent-safe. So I considered extending it to make in-application communication more flexible. Below is what provided by this embeddable piece of software.

✌️ | Producer | Consumer
--- | --: | --:
Single | ✅ | ✅
Multiple | ✅ | ✅

## Design

![architecture](./sc/architecture.jpg)

## Usage

First create a Go project with **GOMOD** support.

```bash
mkdir test_pubsub

cd test_pubsub
go mod init github.com/<username>/test_pubsub

touch main.go
```

Now add `github.com/itzmeanjan/pubsub` as your project dependency

```bash
go get github.com/itzmeanjan/pubsub # v0.1.0 latest
```

And follow full [example](./example/main.go).

You can check package documentation [here](https://pkg.go.dev/github.com/itzmeanjan/pubsub)
