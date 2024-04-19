package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

const (
	hostPort = "localhost:12345"
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

	err = pingMessage.Send(ws, nil)
	fmt.Printf("err: %v\n", err)

	go func() {
		defer func() {
			if r := recover(); r != nil {
				fmt.Println("panic recovered: ", r)
			}
		}()

		for range time.Tick(3 * time.Second) {
			if err := pingMessage.Send(ws, nil); err != nil {
				log.Fatal(err)
			}
		}
	}()

	for {
		if _, err := ws.Write([]byte("test subscribe")); err != nil {
			log.Fatal(err)
		}

		fmt.Println("---------- read ----------")
		ws.SetReadDeadline(time.Now().Add(10 * time.Second))
		var msg = make([]byte, 512)
		var n int
		if n, err = ws.Read(msg); err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				fmt.Println("read timeout")

				continue
			}

			log.Fatal(err)
		}
		fmt.Printf("Received: %s.\n", msg[:n])

		time.Sleep(5 * time.Second)
	}
}
