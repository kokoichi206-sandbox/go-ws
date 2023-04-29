package main

import (
	"fmt"
	"math/rand"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{}

func subscribe(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Printf("err: %v\n", err)

		return
	}
	defer c.Close()

	for range time.Tick(3 * time.Minute) {
		time.Sleep(time.Duration(rand.Intn(1000)) * time.Millisecond)

		if err := c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("hello %v", time.Now()))); err != nil {
			fmt.Printf("err: %v\n", err)

			break
		}
	}
}

func main() {
	http.HandleFunc("/subscribe", subscribe)
	http.ListenAndServe(":11111", nil)
}
