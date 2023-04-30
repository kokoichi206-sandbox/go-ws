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

// Ping must be called concurrently with Reader
func Ping(c *websocket.Conn, ctx context.Context) {
	pingInterval := time.Second * 30

	for {
		pingErr := c.Ping(ctx)

		if pingErr == nil {
			time.Sleep(pingInterval)
		} else {
			fmt.Printf("pingErr: %v\n", pingErr)

			return
		}
	}
}

func wsAccessGoroutine(ctx context.Context, w io.Writer) {
	c, _, err := websocket.Dial(ctx, "ws://localhost:11111/subscribe", nil)
	if err != nil {
		fmt.Printf("err: %v\n", err)

		return
	}
	defer c.Close(websocket.StatusInternalError, "the sky is falling")

	defer fmt.Println("wsAccessD defer")

	// Q. How to ping properly?
	go Ping(c, ctx)

	errChan := make(chan error)
	defer close(errChan)

	go func() {
		for {
			fmt.Printf("read")
			// context support!
			mType, body, err := c.Read(ctx)

			switch websocket.CloseStatus(err) {
			case websocket.StatusNormalClosure:
				fmt.Println("websocket.StatusNormalClosure")
				errChan <- fmt.Errorf("websocket connection is closed by server.")

				return
			}

			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				fmt.Println("netErr.Timeout()")
				errChan <- fmt.Errorf("websocket connection timeout.")

				continue
			}

			if errors.Is(err, context.Canceled) {
				// This is different from gorilla websocket.
				fmt.Println("child context.Canceled in c.Read!")
				// no need to pass errChan? (parent will be also closed)

				return
			}

			if errors.Is(err, context.DeadlineExceeded) {
				fmt.Println("context.DeadlineExceeded in c.Read!")

				// go to next read loop, but the connection is already closed,
				// so, it will be failed in the next loop.
				continue
			}

			if err != nil {
				fmt.Printf("err!!!! : %v\n", err)

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
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("parent context.Canceled")

			return
		case rErr := <-errChan:
			fmt.Printf("rErr: %v\n", rErr)

			return
		}
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

	// Q. How to ping properly?
	go Ping(c, ctx)

	for {
		// No need to check context.Canceled here!!!

		// select {
		// case <-ctx.Done():
		// 	fmt.Println("context.Canceled")

		// 	return

		// default:
		// }

		fmt.Printf("\"read\": %v\n", "read")

		// timeout (or deadline) will destroy the connection itself.
		// so, it's not what I want (but it can be covered by goroutine + context.Cancel)
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		// ctx, cancel := context.WithDeadline(ctx, time.Now().Add(5*time.Second))

		mType, body, err := c.Read(ctx)

		switch websocket.CloseStatus(err) {
		case websocket.StatusNormalClosure:
			fmt.Println("websocket.StatusNormalClosure")
			cancel()

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
			cancel()

			return
		}

		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Println("context.DeadlineExceeded in c.Read!")
			cancel()

			// go to next read loop, but the connection is closed already,
			// so, it will be failed in the next loop.
			continue
		}

		if err != nil {
			fmt.Printf("err!!!! : %v\n", err)
			cancel()

			return
		}

		cancel()

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

func goroutine(w http.ResponseWriter, r *http.Request) {
	wsAccessGoroutine(r.Context(), w)
}

func main() {
	go checkGoroutineNum()

	http.HandleFunc("/goroutine", goroutine)
	http.HandleFunc("/deadline", deadline)
	http.ListenAndServe(":7776", nil)
}
