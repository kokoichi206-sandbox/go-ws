package main

import (
	"context"
	"encoding/json"
	"errors"
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

// PingFrame 送信のための Codec。
// see: https://github.com/golang/net/blob/v0.24.0/websocket/websocket.go#L372-L419
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

	// output はメッセージを表示するための io.Writer。
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

	ctx, cancel := context.WithCancelCause(context.Background())
	defer cancel(nil)

	// Ping するための goroutine。
	go func(ctx context.Context) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("[ping] panic recovered", r)
			}
		}()

		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := pingMessage.Send(ws, nil); err != nil {
					slog.Error("pingMessage.Send", err)
					return
				}
			}
		}
	}(ctx)

	// Read するための goroutine。
	go func(ctx context.Context, cancel context.CancelCauseFunc) {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("[read] panic recovered", r)
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			fr, err := ws.NewFrameReader()
			if err != nil {
				slog.Error("ws.NewFrameReader", err)

				return
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

			case websocket.CloseFrame:
				slog.Info("CloseFrame received")
				cancel(errors.New("CloseFrame received"))
				return

			default:
			}

			// 不要な fr を読み捨てる。
			io.Copy(io.Discard, fr)
		}
	}(ctx, cancel)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if _, err := ws.Write([]byte(fmt.Sprintf("hello im %s", c.name))); err != nil {
			slog.Error("ws.Write", err)
			return fmt.Errorf("failed to ws.Write: %w", err)
		}

		// メッセージ送信のエミュレーション。
		// ランダムな時間待機してから再度メッセージを送信する。
		time.Sleep(time.Duration((rand.IntN(5) + 1)) * time.Second)
	}
}

func main() {
	// flag の設定。
	hostPort := flag.String("hostPort", defaultHostPort, "Host and port of the server")
	topic := flag.String("topic", "topic", "The topic to subscribe to")
	logLevel := flag.String("logLevel", defaultLogLevel.String(), "The log level")
	name := flag.String("name", "john doe", "The name of the client")
	flag.Parse()

	// logger の設定。
	ll := defaultLogLevel
	ll.UnmarshalText([]byte(*logLevel))
	slog.SetLogLoggerLevel(ll)

	// client の作成と実行。
	cl := newClient(*hostPort, *topic, *name)
	cl.run()
}
