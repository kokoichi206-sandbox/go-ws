package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"log/slog"
	"math/rand/v2"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

const (
	defaultHostPort = "localhost:12345"
	pingInterval    = 3 * time.Second
	defaultLogLevel = slog.LevelInfo
)

var pingMessage = websocket.Codec{
	Marshal:   marshal,
	Unmarshal: unmarshal,
}

func marshal(_ any) (msg []byte, payloadType byte, err error) {
	return []byte("ping"), websocket.PingFrame, nil
}

func unmarshal(msg []byte, payloadType byte, v any) (err error) {
	return json.Unmarshal(msg, v)
}

type client struct {
	hostPort string
	topic    string
	name     string

	// received messages are written to this writer
	output io.Writer
}

func newClient(hostPort, topic, name string) *client {
	return &client{
		hostPort: hostPort,
		topic:    topic,
		name:     name,

		output: os.Stdout,
	}
}

func (c *client) run() error {
	origin := fmt.Sprintf("http://%s", c.hostPort)
	url := fmt.Sprintf("ws://%s/%s", c.hostPort, c.topic)

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	// Send ping messages to the server.
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", r)
			}
		}()

		for range time.Tick(pingInterval) {
			if err := pingMessage.Send(ws, nil); err != nil {
				slog.Error("pingMessage.Send", err)
				return
			}
		}
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic recovered", r)
			}
		}()

		for {
			fr, err := ws.NewFrameReader()
			if err != nil {
				slog.Error("ws.NewFrameReader", err)
			}

			switch fr.PayloadType() {
			case websocket.PongFrame:
				b, _ := io.ReadAll(fr)
				slog.Debug(fmt.Sprintf("PongFrame: %s", string(b)))
				continue

			case websocket.TextFrame:
				b, _ := io.ReadAll(fr)
				fmt.Fprintf(c.output, "%s\n", string(b))
				continue
			}

			io.Copy(io.Discard, fr)
		}
	}()

	for {
		if _, err := ws.Write([]byte(fmt.Sprintf("hello im %s", c.name))); err != nil {
			log.Fatal(err)
		}

		// Send messages with random interval.
		time.Sleep(time.Duration((rand.IntN(5) + 1)) * time.Second)
	}
}

func main() {
	hostPort := flag.String("hostPort", defaultHostPort, "Host and port of the server")
	topic := flag.String("topic", "topic", "The topic to subscribe to")
	logLevel := flag.String("logLevel", defaultLogLevel.String(), "The log level")
	name := flag.String("name", "john doe", "The name of the client")

	flag.Parse()

	ll := defaultLogLevel
	ll.UnmarshalText([]byte(*logLevel))
	slog.SetLogLoggerLevel(ll)

	cl := newClient(*hostPort, *topic, *name)
	cl.run()
}
