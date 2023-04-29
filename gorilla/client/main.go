package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"runtime"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

func checkGoroutineNum() {
	for range time.Tick(5 * time.Second) {
		fmt.Printf("runtime.NumGoroutine(): %v\n", runtime.NumGoroutine())
	}
}

// There's a memory leak, how can I fix it?
func wsAccessG(ctx context.Context, w io.Writer) {
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws://localhost:11111/subscribe", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	defer conn.Close()

	fmt.Printf("resp.StatusCode: %v\n", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("body: %v\n", body)

	errChan := make(chan error, 1)
	defer close(errChan)

	mu := sync.RWMutex{}

	go func() {
		defer func() {
			if err := recover(); err != nil {
				fmt.Println("recover: ", err)
			}
		}()

		mu.RLock()
		defer mu.RUnlock()

		for {
			mType, body, err := conn.ReadMessage()
			if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
				errChan <- nil

				return
			}

			if err != nil {
				errChan <- err

				return
			}

			switch mType {
			case websocket.TextMessage:
				fmt.Printf("TextMessage: %v\n", string(body))
				w.Write(body)

			default:
				errChan <- fmt.Errorf("unknown message type: %v", mType)

				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context.Canceled")

			return

		case err := <-errChan:
			fmt.Println("err: %v", err)

			return

		default:
		}
	}
}

func goroutine(w http.ResponseWriter, r *http.Request) {
	wsAccessG(r.Context(), w)
}

func wsAccess(ctx context.Context, w io.Writer) {
	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, "ws://localhost:11111/subscribe", nil)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	defer conn.Close()

	fmt.Printf("resp.StatusCode: %v\n", resp.StatusCode)
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("body: %v\n", body)

	for {
		select {
		case <-ctx.Done():
			fmt.Println("context.Canceled")

			return

		default:
		}

		messageType, body, err := conn.ReadMessage()
		if err != nil {
			log.Fatal(err)
		}

		switch messageType {
		case websocket.TextMessage:
			fmt.Printf("TextMessage: %v\n", string(body))
			w.Write(body)

		default:
			fmt.Println("unknown message type: ", messageType)
			return
		}
	}
}

func subscribe(w http.ResponseWriter, r *http.Request) {
	wsAccess(r.Context(), w)
}

func main() {
	go checkGoroutineNum()

	http.HandleFunc("/subscribe", subscribe)
	http.HandleFunc("/goroutine", goroutine)
	http.ListenAndServe(":7999", nil)
}
