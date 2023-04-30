package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"golang.org/x/net/websocket"
)

func main() {
	origin := "http://localhost:11111"
	url := "ws://localhost:11111/subscribe"
	// url := "ws://localhost:11111"
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	for {
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
	}
}
