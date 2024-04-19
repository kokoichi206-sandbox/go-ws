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
	hostPort     = "localhost:12341"
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

func main() {
	origin := fmt.Sprintf("http://%s", hostPort)
	url := fmt.Sprintf("ws://%s/subscribe", hostPort)

	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	go func() {
		fmt.Println("---------- write ping  ----------")

		defer func() {
			if r := recover(); r != nil {
				fmt.Println("panic recovered: ", r)
			}
		}()

		for range time.Tick(pingInterval) {
			if err := pingMessage.Send(ws, nil); err != nil {
				log.Fatal(err)
			}
		}
	}()

	go func() {
		fmt.Println("---------- read loop ----------")

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
				// fmt.Printf("Received: %s.\n", string(msg))
			}
		}
	}()

	for {
		if _, err := ws.Write([]byte("test subscribe")); err != nil {
			log.Fatal(err)
		}

		time.Sleep(5 * time.Second)
	}
}
