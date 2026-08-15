Using go1.26.0

## Run

### Simple TCP server (goroutine per connection)

```bash
go run ./cmd/tcp-server
```

### TCP server with worker pool

```bash
go run ./cmd/thread-pool-server
```

### TCP server with I/O multiplexing (epoll)

```bash
go run ./cmd/io-multiplexing-server
```

Note: I/O multiplexing mode uses epoll and is only supported on Linux.
If not on Linux, uses

```bash
docker compose run --rm go go run ./cmd/io-multiplexing-server
```
