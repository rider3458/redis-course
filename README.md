Using go1.26.0

## Run

### Simple TCP server (goroutine per connection)

```bash
make run-tcp
```

### TCP server with worker pool

```bash
make run-thread-pool
```

### TCP server with I/O multiplexing (epoll)

```bash
make run-io-multiplexing
```

### RESP server

```bash
make run-resp
```

Note: I/O multiplexing mode uses epoll and is only supported on Linux.
If not on Linux, use

```bash
make run-io-multiplexing-docker
```
