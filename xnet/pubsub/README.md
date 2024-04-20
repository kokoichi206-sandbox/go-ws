# topic ごとの pubsub 実装

## 仕様

- クライアントは ws で接続する
- topic は path で表す
  - `/{topic}`
- 全クライアントは pub & sub で接続する

## 動作確認

### 1. サーバーを起動する

``` sh
$ ls
README.md client    memo.md   server

$ cd server
$ go run main.go

# 詳細なログを出したい時。
go run main.go -logLevel=debug
```

### 2. 複数のクライアントを起動

``` sh
$ ls
README.md client    memo.md   server

$ cd client

# client 1
go run main.go -name=minami
# client 2
go run main.go -name=pien
# client 3 (他の topic に接続)
go run main.go -name=pien -topic=tigau

# 詳細なログを出したい時。
go run main.go -name=minami -logLevel=debug
```
