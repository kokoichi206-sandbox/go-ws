package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"sync"
	"syscall"

	"golang.org/x/net/websocket"
)

const (
	hostPort        = ":12345"
	defaultLogLevel = slog.LevelInfo
)

var pongMessage = websocket.Codec{
	Marshal:   marshal,
	Unmarshal: unmarshal,
}

func marshal(_ any) (msg []byte, payloadType byte, err error) {
	return []byte("thanks to ping!"), websocket.PongFrame, nil
}

func unmarshal(msg []byte, payloadType byte, v any) (err error) {
	return json.Unmarshal(msg, v)
}

type textFR interface {
	io.Reader
	Len() int
}

type handler struct {
	topics map[string][]*websocket.Conn
	mu     sync.RWMutex
}

func (h *handler) getConns(topic string) []*websocket.Conn {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return slices.Clone(h.topics[topic])
}

func (h *handler) join(topic string, ws *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.topics[topic] = append(h.topics[topic], ws)
}

func (h *handler) leave(topic string, ws *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	conns := h.topics[topic]
	for i, conn := range conns {
		if conn == ws {
			h.topics[topic] = slices.Delete(conns, i, i+1)
			return
		}
	}
}

func (h *handler) pubsub(ws *websocket.Conn) {
	defer ws.Close()

	topic := ws.Request().PathValue("topic")
	h.join(topic, ws)
	defer h.leave(topic, ws)

	for {
		r, err := ws.NewFrameReader()
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Info("connection closed")
				break
			}

			continue
		}

		switch r.PayloadType() {
		case websocket.PingFrame:
			// not read payload
			pongMessage.Send(ws, nil)
			continue

		case websocket.TextFrame:
			if err := h.handleTextFrame(r, topic, ws); err != nil {
				slog.Error(fmt.Sprintf("failed to handle text frame: %s", err))
				continue
			}

		default:
		}

		io.Copy(io.Discard, r)
	}
}

func (h *handler) handleTextFrame(r textFR, topic string, ws *websocket.Conn) error {
	if r.Len() > 1998_0206 {
		return fmt.Errorf("too large payload: %d", r.Len())
	}

	res, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("failed to read payload: %w", err)
	}

	if err := h.publishText(topic, res, ws); err != nil {
		return fmt.Errorf("failed to publish: %w", err)
	}

	return nil
}

func (h *handler) publishText(topic string, payload []byte, publisher *websocket.Conn) error {
	conns := h.getConns(topic)

	slog.Debug(fmt.Sprintf("len(conns): %v", len(conns)))
	slog.Debug(fmt.Sprintf("string(payload): %v\n", string(payload)))

	for _, conn := range conns {
		if conn != publisher {
			if err := websocket.Message.Send(conn, string(payload)); err != nil {
				return fmt.Errorf("failed to send message: %w", err)
			}
		}
	}

	return nil
}

func (h *handler) close() {
	for topic, conns := range h.topics {
		for _, conn := range conns {
			conn.Close()
		}

		delete(h.topics, topic)
	}
}

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)
	logLevel := flag.String("logLevel", defaultLogLevel.String(), "The log level")
	flag.Parse()

	ll := defaultLogLevel
	ll.UnmarshalText([]byte(*logLevel))
	slog.SetLogLoggerLevel(ll)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt, os.Kill)
	defer stop()

	h := &handler{
		topics: make(map[string][]*websocket.Conn),
		mu:     sync.RWMutex{},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /{topic}", websocket.Handler(h.pubsub))

	srv := &http.Server{
		Addr:    hostPort,
		Handler: mux,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			if errors.Is(err, http.ErrServerClosed) {
				slog.Info("server closed gracefully")
				return
			}
			slog.Error(fmt.Sprintf("failed to listen and serve: %s", err))
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down...")
	h.close()
	srv.Shutdown(ctx)
}
