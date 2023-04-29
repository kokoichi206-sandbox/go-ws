package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"runtime"
	"time"

	"nhooyr.io/websocket"
)

func checkGoroutineNum() {
	for range time.Tick(5 * time.Second) {
		fmt.Printf("runtime.NumGoroutine(): %v\n", runtime.NumGoroutine())
	}
}

func wsAccessD(ctx context.Context, w io.Writer) {
	c, _, err := websocket.Dial(ctx, "ws://localhost:11111/subscribe", nil)
	if err != nil {
		fmt.Printf("err: %v\n", err)

		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	defer fmt.Println("wsAccessD defer")

	for {
		// No need to check context.Canceled here!!!

		// select {
		// case <-ctx.Done():
		// 	fmt.Println("context.Canceled")

		// 	return

		// default:
		// }

		fmt.Printf("\"read\": %v\n", "read")

		// but, timeout will destroy the connection itself.
		// so, it's not good...
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

		mType, body, err := c.Read(ctx)
		fmt.Printf("err: %v\n", err)

		switch websocket.CloseStatus(err) {
		case websocket.StatusNormalClosure:
			fmt.Println("websocket.StatusNormalClosure")

			return
		}

		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			fmt.Println("netErr.Timeout()")

			cancel()

			continue
		}

		if errors.Is(err, context.Canceled) {
			// This is different from gorilla websocket.
			fmt.Println("context.Canceled in c.Read!")

			return
		}

		switch mType {
		case websocket.MessageText:
			fmt.Printf("TextMessage: %v\n", string(body))
			w.Write(body)

		default:
			fmt.Println("unknown message type: ", mType)

			return
		}
	}
}

func deadline(w http.ResponseWriter, r *http.Request) {
	wsAccessD(r.Context(), w)
}

func main() {
	go checkGoroutineNum()

	http.HandleFunc("/deadline", deadline)
	http.ListenAndServe(":7776", nil)
}
