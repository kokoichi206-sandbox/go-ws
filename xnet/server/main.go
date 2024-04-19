package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"

	"golang.org/x/net/websocket"
)

const (
	hostPort = ":12341"
)

var pongMessage = websocket.Codec{
	Marshal:   marshal,
	Unmarshal: unmarshal,
}

func marshal(v any) (msg []byte, payloadType byte, err error) {
	return []byte("thanks to ping!"), websocket.PongFrame, nil
}

func unmarshal(msg []byte, payloadType byte, v any) (err error) {
	return json.Unmarshal(msg, v)
}

type textFR interface {
	io.Reader
	Len() int
}

func subscribe(ws *websocket.Conn) {
	defer ws.Close()

	for {
		r, err := ws.NewFrameReader()
		if err != nil {
			if errors.Is(err, io.EOF) {
				fmt.Printf("connection closed\n")
				break
			}

			fmt.Printf("unexpected err: %v\n", err)
		}

		switch r.PayloadType() {
		case websocket.PingFrame:
			b, _ := io.ReadAll(r)
			fmt.Printf("string(b): %v\n", string(b))
			pongMessage.Send(ws, nil)
			continue

		case websocket.TextFrame:
			handleTextFrame(r, ws)
		}
	}
}

func handleTextFrame(r textFR, ws *websocket.Conn) error {
	if r.Len() > 1998_0206 {
		return fmt.Errorf("too large payload: %d", r.Len())
	}

	res, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read payload: %w", err)
	}

	fmt.Printf("received: %s\n", res)
	ws.Write(res)

	return nil
}

func main() {
	http.Handle("/subscribe", websocket.Handler(subscribe))
	if err := http.ListenAndServe(hostPort, nil); err != nil {
		panic("ListenAndServe: " + err.Error())
	}
}
