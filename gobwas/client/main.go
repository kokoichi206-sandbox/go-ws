package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/gobwas/ws"
	"github.com/gobwas/ws/wsutil"
)

func checkGoroutineNum() {
	for range time.Tick(5 * time.Second) {
		fmt.Printf("runtime.NumGoroutine(): %v\n", runtime.NumGoroutine())
	}
}

func gobwasTest(ctx context.Context) error {
	conn, r, hs, err := ws.DefaultDialer.Dial(ctx, "ws://localhost:11111/subscribe")
	// ws.Dial(ctx, "ws://localhost:11111/subscribe")

	fmt.Printf("conn: %v\n", conn)
	fmt.Printf("r: %v\n", r)
	fmt.Printf("hs: %v\n", hs)
	fmt.Printf("err: %v\n", err)

	pingInterval := time.Second * 5
	ticker := time.NewTicker(pingInterval)

	for {
		select {
		case <-ctx.Done():
			return context.Canceled
		case <-ticker.C:
			fmt.Println("ping!!!")
			conn.Write(ws.CompiledPing)
		default:
		}

		fmt.Println("read")
		conn.SetReadDeadline(time.Now().Add(4 * time.Second))
		// ---------- low level read ----------
		// var msg = make([]byte, 512)
		// n, err := conn.Read(msg)
		// if err != nil {
		// 	fmt.Printf("err: %v\n", err)
		// }

		body, err := wsutil.ReadServerText(conn)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				fmt.Println("continue")
				continue
			}

			return fmt.Errorf("failed to read message: %w", err)
		}

		fmt.Printf("body: %v\n", string(body))
	}
}

func subscribe(w http.ResponseWriter, r *http.Request) {
	gobwasTest(r.Context())
}

func main() {
	go checkGoroutineNum()

	http.HandleFunc("/subscribe", subscribe)
	http.ListenAndServe(":8181", nil)
}
