## packages

- https://github.com/gorilla/websocket
  - stars: 
    - 19.1k
  - archived
  - BSD-2-Clause license
  - releases: 
    - 8
    - 2022/01/04 1.5.0
  - contributors:
    - 74
- https://github.com/nhooyr/websocket
  - stars: 
    - 2.9k
  - releases: 
    - 33
    - 2021/04/08 1.8.7
    - 2020/05/10 1.8.6
  - contributors:
    - 9
  - full context support
  - MIT license
- https://github.com/gobwas/ws
  - stars: 
    - 5.4k
  - releases: 
    - 14
    - 2023/05/xx 1.3.0
    - 2023/04/04 1.2.0
    - 2021/07/11 1.1.0
  - contributors:
    - 18
  - it fixes 'read timeout' problem
  - MIT license
- https://pkg.go.dev/golang.org/x/net/websocket
  - one of the sub-repositories
  - but it's deprecated ?
    - https://github.com/golang/go/issues/18152
  - BSD-3-Clause

### comparison (2023/4/30)

| | [x/net](https://pkg.go.dev/golang.org/x/net/websocket) | [gorilla](https://github.com/gorilla/websocket) | [nhooyr](https://github.com/nhooyr/websocket) | [gobwas](https://github.com/gobwas/ws) |
| :---: | :---: | :---: | :---: | :---: |
| maintained? | ×? | × | ○ | ○ |
| stars | - | 19.1k | 2.9k | 5.4k |
| releases | ? | [2022/01/04](https://pkg.go.dev/github.com/gorilla/websocket?tab=versions) | [2021/04/08](https://pkg.go.dev/nhooyr.io/websocket?tab=versions) | [2023/04/04](https://pkg.go.dev/github.com/gobwas/ws?tab=versions) |
| version | 0.9.0? | 1.5.0 | 1.8.7 | 1.2.0 |
| contributors | - | 74 | 9 | 18 |
| License | BSD-3 | BSD-2 | MIT | MIT |

**Feat**  
(ref: [websocket#comparison(@nhooyr)](https://github.com/nhooyr/websocket/tree/14fb98eba64eeb5e9d06a88b98c47ae924ac82b4#comparison))

| | [x/net](https://pkg.go.dev/golang.org/x/net/websocket) | [gorilla](https://github.com/gorilla/websocket) | [nhooyr](https://github.com/nhooyr/websocket) | [gobwas](https://github.com/gobwas/ws) |
| :---: | :---: | :---: | :---: | :---: |
| context (rw) | × | × | ○ | × |
| connection alive </br> when deadline </br> (set deadilne) | ○ | × | × | ○ |


## read timeout

https://github.com/gorilla/websocket/issues/474

- if the deadline exceeded, the entire "tcp" connection is destroyed.
  - not only the read (write) connection
    - gorilla
    - nhooyr

## Question

- why the behavior of "read timeout" is different?
- MASK in websocket ?
  - https://www.rfc-editor.org/rfc/rfc6455#section-5.3
- epoll ?
  - https://man7.org/linux/man-pages/man7/epoll.7.html

## Links

- gobwas author's [blog post](https://www.freecodecamp.org/news/million-websockets-and-go-cc58418460bb)
- [websocket#comparison(@nhooyr)](https://github.com/nhooyr/websocket/tree/14fb98eba64eeb5e9d06a88b98c47ae924ac82b4#comparison)
