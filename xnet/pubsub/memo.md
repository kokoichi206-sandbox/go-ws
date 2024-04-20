## 文字化けのようになってしまう

``` sh
$ go run main.go -name=pien
hello im minami
hello im ore
hello im ore
ɉ��     L��`"�����j����������������Ί���87��H^�����$E.�A)B�,C�K7K
```

全ての frameReader を読み取ってなかったことが原因？

読み取らないとなぜそうなってしまうのか。

### 解消方法

[frameReader を必ず閉じるようにした](https://github.com/kokoichi206-sandbox/go-ws/commit/09c7976a14ab135b20e87a0aa42bd6e704d28cc3)

### 原因

frameReader を生成するために、`websocket.Conn` の `ws.NewFrameReader` を使っている。  
（PingFrame 等が読み取れないため conn.Read を使っていない。）

``` go
func (h *handler) pubsub(ws *websocket.Conn) {
	...
	for {
		r, err := ws.NewFrameReader()
		switch r.PayloadType() {
		case websocket.PingFrame:
			b, _ := io.ReadAll(r)
```

NewFrameReader は `websocket.Conn` に埋め込まれている [frameReaderFactory](https://github.com/golang/net/blob/v0.24.0/websocket/websocket.go#L171) が担っている。

``` go
type Conn struct {
	...
	frameReaderFactory
}
// frameReaderFactory is an interface to creates new frame reader.
type frameReaderFactory interface {
	NewFrameReader() (r frameReader, err error)
}
```

この interface の実装は [hybiFrameReaderFactory](https://github.com/golang/net/blob/v0.24.0/websocket/hybi.go#L110-L112) が[担当している](https://github.com/golang/net/blob/v0.24.0/websocket/hybi.go#L342)。

NewFrameReader の生成する FrameReader の reader には [hybiFrameReaderFactory のもつ buffer が使いまわされて](https://github.com/golang/net/blob/v0.24.0/websocket/hybi.go#L174)いる。

``` go
type hybiFrameReaderFactory struct {
	*bufio.Reader
}
func (buf hybiFrameReaderFactory) NewFrameReader() (frame frameReader, err error) {
	hybiFrame := new(hybiFrameReader)
	frame = hybiFrame
	...
	hybiFrame.reader = io.LimitReader(buf.Reader, hybiFrame.header.Length)
	...
}
```

きちんと読み込まないと、後々不整合が起こる（逆に読み込んだら不整合は起こらないのか）。

ドキュメント等にそのような記載がないか、もう一度確認する。
