package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"time"

	"golang.org/x/net/websocket"
)

const (
	hostPort     = "localhost:12345"
	pingInterval = 3 * time.Second
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

func newClient(topic string) {
	origin := fmt.Sprintf("http://%s", hostPort)
	url := fmt.Sprintf("ws://%s/%s", hostPort, topic)

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("panic recovered: ", r)
			}
		}()

		for range time.Tick(pingInterval) {
			if err := pingMessage.Send(ws, nil); err != nil {
				fmt.Printf("err: %v\n", err)
				return
			}
		}
	}()

	go func() {
		for {
			fr, err := ws.NewFrameReader()
			if err != nil {
				log.Fatal(err)
			}

			switch fr.PayloadType() {
			case websocket.PongFrame:
				b, _ := io.ReadAll(fr)
				fmt.Printf("PongFrame string(b): %v\n", string(b))
				continue

			case websocket.TextFrame:
				b, _ := io.ReadAll(fr)
				fmt.Printf("string(b): %v\n", string(b))
				continue
			}
		}
	}()

	for {
		if _, err := ws.Write([]byte("new kawaii")); err != nil {
			log.Fatal(err)
		}

		time.Sleep(5 * time.Second)
	}
}

func main() {
	go newClient("topic")
	time.Sleep(1 * time.Second)
	go newClient("topic")

	go newClient("topic2")
	time.Sleep(1 * time.Second)
	go newClient("topic2")

	time.Sleep(100 * time.Second)
}
