package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

// I think this is not correct...
// cannot read ping message from client.
func pingReceived(c *websocket.Conn) {
	for {
		fmt.Printf("\"ReadMessage\": %v\n", "ReadMessage")
		mType, _, err := c.ReadMessage()
		if err != nil {
			fmt.Printf("err: %v\n", err)

			break
		}

		switch mType {
		case websocket.PingMessage:
			fmt.Printf("ping received")
		default:
			fmt.Printf("mType: %v\n", mType)

			break
		}
	}
}

func subscribe(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("err: %v\n", err)

		return
	}
	defer c.Close()

	go pingReceived(c)

	c.SetPingHandler(func(appData string) error {
		fmt.Printf("ping received!  appData: %v\n", appData)
		return nil
	})

	for range time.Tick(15 * time.Second) {
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		if err := c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("hello %v", time.Now()))); err != nil {
			fmt.Printf("err: %v\n", err)

			break
		}

		// c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}
}

func main() {
	http.HandleFunc("/subscribe", subscribe)
	http.ListenAndServe(":11111", nil)
}
