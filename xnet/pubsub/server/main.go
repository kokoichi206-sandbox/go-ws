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

// PongFrame 送信のための Codec。
// see: https://github.com/golang/net/blob/v0.24.0/websocket/websocket.go#L372-L419
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
	// topic ごとのコネクション一覧。
	// コネクションの識別は保持するポインタの値比較で行う。
	topics   map[string][]*websocket.Conn
	topicsMu sync.RWMutex
}

// getConns は topic に紐づくコネクション一覧を返す。
//
// 注意)
//   - []*websocket.Conn の各要素の値が変わってしまうことまでは防げない。
func (h *handler) getConns(topic string) []*websocket.Conn {
	h.topicsMu.RLock()
	defer h.topicsMu.RUnlock()

	return slices.Clone(h.topics[topic])
}

// join は topic にコネクションを追加する。
func (h *handler) join(topic string, ws *websocket.Conn) {
	h.topicsMu.Lock()
	defer h.topicsMu.Unlock()

	h.topics[topic] = append(h.topics[topic], ws)
}

// leave は topic からコネクションを削除する。
func (h *handler) leave(topic string, ws *websocket.Conn) {
	h.topicsMu.Lock()
	defer h.topicsMu.Unlock()

	conns := h.topics[topic]
	for i, conn := range conns {
		if conn == ws {
			h.topics[topic] = slices.Delete(conns, i, i+1)
			return
		}
	}
}

// pubsub は WebSocket での pubsub を行う。
func (h *handler) pubsub(ws *websocket.Conn) {
	defer ws.Close()

	topic := ws.Request().PathValue("topic")
	h.join(topic, ws)
	defer h.leave(topic, ws)

	for {
		// fr は最後まで読み込む必要がある。
		fr, err := ws.NewFrameReader()
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Info("connection closed")
				break
			}

			continue
		}

		switch fr.PayloadType() {
		case websocket.PingFrame:
			b, _ := io.ReadAll(fr)
			slog.Debug(fmt.Sprintf("PingFrame: %s", string(b)))
			pongMessage.Send(ws, nil)
			continue

		case websocket.TextFrame:
			if err := h.handleTextFrame(fr, topic, ws); err != nil {
				slog.Error(fmt.Sprintf("failed to handle text frame: %s", err))
				continue
			}

		default:
		}

		// 不要な fr を読み捨てる。
		io.Copy(io.Discard, fr)
	}
}

// handleTextFrame は TextFrame を処理する。
//
// 仕様:
//
//	TextFrame のペイロードが大きすぎる場合はエラーを返す。
//	それ以外の場合は subscribe している topic にメッセージを送信する。
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

// close は handler のリソースを解放する。
func (h *handler) close() {
	for topic, conns := range h.topics {
		for _, conn := range conns {
			conn.Close()
		}

		delete(h.topics, topic)
	}
}

func main() {
	// flag の設定。
	slog.SetLogLoggerLevel(slog.LevelDebug)
	logLevel := flag.String("logLevel", defaultLogLevel.String(), "The log level")
	flag.Parse()

	// logger の設定。
	ll := defaultLogLevel
	ll.UnmarshalText([]byte(*logLevel))
	slog.SetLogLoggerLevel(ll)

	// handler の設定。
	h := &handler{
		topics:   make(map[string][]*websocket.Conn),
		topicsMu: sync.RWMutex{},
	}

	mux := http.NewServeMux()
	mux.Handle("GET /{topic}", websocket.Handler(h.pubsub))

	srv := &http.Server{
		Addr:    hostPort,
		Handler: mux,
	}

	// graceful shutdown の準備
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGTERM, os.Interrupt, os.Kill)
	defer stop()

	// signal を受け取るために goroutine で ListenAndServe を実行する。
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

	// 正常にリソースを解放した後、サーバーをシャットダウンする。
	h.close()
	if err := srv.Shutdown(ctx); err != nil {
		slog.Error(fmt.Sprintf("failed to shutdown: %s", err))
	}
}
