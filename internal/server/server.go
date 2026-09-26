package server

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/rider3458/redis-course/internal/io_multiplexing"
	"github.com/rider3458/redis-course/internal/protocol"
)

const (
	defaultShutdownTimeout = 5 * time.Second
	handlerDrainGrace      = time.Second
	drainPollInterval      = 5 * time.Millisecond
)

type Server struct {
	config ServerConfig
}

type ServerConfig struct {
	Address              string
	UseIOMultiplexing    bool
	UseRESPProtocol      bool
	MaxConnections       int
	IsPoolEnabled        bool
	PoolSize             int
	ConnectionsPerWorker int
	QueueSize            int
	ShutdownTimeout      time.Duration
}

func New(config ServerConfig) *Server {
	return &Server{config}
}

func (s *Server) handleConnection(conn net.Conn, tracked *trackedConn) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	buffer := make([]byte, 1024)
	pending := make([]byte, 0)

	for {
		// Blocked in Read means no command is in flight, so a shutdown
		// drain may close this connection immediately.
		tracked.setBusy(false)
		n, err := conn.Read(buffer)

		if n > 0 {
			tracked.setBusy(true)
			pending = append(pending, buffer[:n]...)

			if s.config.UseRESPProtocol {
				respErr := s.processRESPBuffer(&pending, func(cmd *protocol.Command) error {
					fmt.Println("Received from", conn.RemoteAddr(), ":", cmd.Cmd, cmd.Args)
					_, writeErr := conn.Write(buildRESPReply(cmd))
					return writeErr
				})
				if respErr != nil {
					fmt.Println("Error when parsing request: ", respErr)
					return
				}
				continue
			}

			for {
				eolIdx := bytes.IndexByte(pending, '\n')
				if eolIdx == -1 {
					break
				}

				request := string(pending[:eolIdx])
				pending = pending[eolIdx+1:]

				if s.config.IsPoolEnabled {
					fmt.Println("Received from", conn.RemoteAddr(), ": ", request)
				}
				fmt.Println("Received from", conn.RemoteAddr(), ": ", request)

				conn.Write([]byte("OK\n"))
			}
		}

		if err != nil {
			if err == io.EOF || errors.Is(err, net.ErrClosed) {
				fmt.Println("Closed connection from ", conn.RemoteAddr())
			} else {
				fmt.Println("Error when reading request: ", err)
			}
			return
		}
	}
}

func (s *Server) Serve() {
	if s.config.UseIOMultiplexing {
		s.serveWithIOMultiplexing()
		return
	}

	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server running on ", s.config.Address)

	if s.config.IsPoolEnabled {
		fmt.Println("Worker Pool Size: ", s.config.PoolSize)
		pool := NewWorkerPool(
			s.config.PoolSize,
			s.config.ConnectionsPerWorker,
			s.config.QueueSize,
			func(conn net.Conn) { s.handleConnection(conn, nil) },
		)
		pool.Start()
		defer pool.Close()
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting:", err)
				continue
			}
			pool.Submit(conn)
		}
	} else {
		for {
			conn, err := listener.Accept()
			if err != nil {
				fmt.Println("Error accepting:", err)
				continue
			}

			go s.handleConnection(conn, nil)
		}
	}

}

// ListenAndServe serves connections until ctx is canceled. On shutdown it
// stops accepting, closes idle connections at once, gives in-flight commands
// up to ShutdownTimeout to finish, then force-closes the rest and waits for
// every handler to return. A nil error means a clean shutdown.
func (s *Server) ListenAndServe(ctx context.Context) error {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		return err
	}
	return s.serveListener(ctx, listener)
}

func (s *Server) serveListener(ctx context.Context, listener net.Listener) error {
	timeout := s.config.ShutdownTimeout
	if timeout <= 0 {
		timeout = defaultShutdownTimeout
	}

	conns := newConnSet()
	var wg sync.WaitGroup

	// Closed by the accept loop on exit so this watcher does not outlive it.
	stopped := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			listener.Close()
		case <-stopped:
		}
	}()
	defer close(stopped)

	fmt.Println("Server running on ", listener.Addr())

	for {
		conn, err := listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				break
			}
			fmt.Println("Error accepting:", err)
			continue
		}

		tracked := &trackedConn{Conn: conn}
		conns.add(tracked)
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer conns.remove(tracked)
			s.handleConnection(tracked, tracked)
		}()
	}

	fmt.Println("Shutting down: draining connections")
	conns.drain(timeout)
	// Force-close wakes blocked handlers, so this wait is only an unwind
	// allowance. Bound it: a handler stuck in a syscall must not hang exit.
	if !waitGroupTimeout(&wg, handlerDrainGrace) {
		fmt.Println("Shutdown timed out waiting for handlers to exit")
	}
	fmt.Println("Server stopped")
	return nil
}

// waitGroupTimeout waits for wg or returns false once timeout elapses.
func waitGroupTimeout(wg *sync.WaitGroup, timeout time.Duration) bool {
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()
	select {
	case <-done:
		return true
	case <-time.After(timeout):
		return false
	}
}

func (s *Server) serveWithIOMultiplexing() {
	listener, err := net.Listen("tcp", s.config.Address)
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		fmt.Println("Expected TCP listener")
		return
	}

	listenerFile, err := tcpListener.File()
	if err != nil {
		fmt.Println("Error getting listener file:", err)
		return
	}
	defer listenerFile.Close()

	maxConnections := s.config.MaxConnections
	if maxConnections <= 0 {
		maxConnections = 128
	}

	multiplexer, err := io_multiplexing.CreateIOMultiplexer(maxConnections)
	if err != nil {
		fmt.Println("Error creating io multiplexer:", err)
		return
	}
	defer multiplexer.Close()

	listenerFD := int(listenerFile.Fd())
	if err := multiplexer.Monitor(io_multiplexing.Event{Fd: listenerFD, Op: io_multiplexing.OpRead}); err != nil {
		fmt.Println("Error monitoring listener:", err)
		return
	}

	pending := make(map[int][]byte)

	fmt.Println("Server running on ", s.config.Address)
	fmt.Println("I/O Multiplexing enabled")

	for {
		events, err := multiplexer.Wait()
		if err != nil {
			fmt.Println("Error waiting for events:", err)
			continue
		}

		for _, event := range events {
			if event.Fd == listenerFD {
				clientFD, _, acceptErr := syscall.Accept(listenerFD)
				if acceptErr != nil {
					fmt.Println("Error accepting:", acceptErr)
					continue
				}

				if len(pending) >= maxConnections {
					fmt.Println("Max connections reached, rejecting fd:", clientFD)
					syscall.Close(clientFD)
					continue
				}

				if err := multiplexer.Monitor(io_multiplexing.Event{Fd: clientFD, Op: io_multiplexing.OpRead}); err != nil {
					fmt.Println("Error monitoring client:", err)
					syscall.Close(clientFD)
					continue
				}

				pending[clientFD] = make([]byte, 0)
				fmt.Println("Client connected, fd:", clientFD)
				continue
			}

			if _, exists := pending[event.Fd]; !exists {
				continue
			}

			buffer := make([]byte, 1024)
			n, readErr := syscall.Read(event.Fd, buffer)
			if n == 0 && readErr == nil {
				readErr = io.EOF
			}

			if n > 0 {
				pending[event.Fd] = append(pending[event.Fd], buffer[:n]...)

				if s.config.UseRESPProtocol {
					pendingBuffer := pending[event.Fd]
					respErr := s.processRESPBuffer(&pendingBuffer, func(cmd *protocol.Command) error {
						fmt.Println("Received from fd", event.Fd, ":", cmd.Cmd, cmd.Args)

						response := buildRESPReply(cmd)
						for len(response) > 0 {
							written, writeErr := syscall.Write(event.Fd, response)
							if writeErr != nil {
								return writeErr
							}
							response = response[written:]
						}
						return nil
					})
					pending[event.Fd] = pendingBuffer
					if respErr != nil {
						readErr = respErr
					}
				} else {

					for {
						eolIdx := bytes.IndexByte(pending[event.Fd], '\n')
						if eolIdx == -1 {
							break
						}

						request := string(pending[event.Fd][:eolIdx])
						pending[event.Fd] = pending[event.Fd][eolIdx+1:]

						fmt.Println("Received from fd", event.Fd, ": ", request)

						response := []byte("OK\n")
						for len(response) > 0 {
							written, writeErr := syscall.Write(event.Fd, response)
							if writeErr != nil {
								fmt.Println("Error when writing response:", writeErr)
								readErr = writeErr
								break
							}
							response = response[written:]
						}

						if readErr != nil {
							break
						}
					}
				}
			}

			if readErr != nil {
				if readErr != io.EOF && readErr != syscall.ECONNRESET && readErr != syscall.EPIPE {
					fmt.Println("Error when reading request:", readErr)
				} else {
					fmt.Println("Closed connection from fd", event.Fd)
				}

				if unmonitorErr := multiplexer.Unmonitor(event.Fd); unmonitorErr != nil {
					fmt.Println("Error unmonitoring client:", unmonitorErr)
				}
				syscall.Close(event.Fd)
				delete(pending, event.Fd)
			}
		}
	}
}

func buildRESPReply(cmd *protocol.Command) []byte {
	return protocol.DispatchCommand(cmd)
}

func (s *Server) processRESPBuffer(pending *[]byte, onCommand func(cmd *protocol.Command) error) error {
	for {
		cmd, consumed, err := protocol.ParseCommand(*pending)
		if err != nil {
			if err == protocol.ErrIncompleteRESP {
				return nil
			}
			return err
		}

		if err := onCommand(cmd); err != nil {
			return err
		}

		*pending = (*pending)[consumed:]
	}
}

// trackedConn marks whether a connection is mid-command so shutdown can tell
// idle connections from ones with work to drain. The nil receiver is a no-op
// so callers that do not track connections can pass nil.
type trackedConn struct {
	net.Conn
	busy atomic.Bool
}

func (tc *trackedConn) setBusy(busy bool) {
	if tc == nil {
		return
	}
	tc.busy.Store(busy)
}

func (tc *trackedConn) isBusy() bool {
	return tc != nil && tc.busy.Load()
}

// connSet tracks open connections during shutdown.
type connSet struct {
	mu    sync.Mutex
	conns map[*trackedConn]struct{}
}

func newConnSet() *connSet {
	return &connSet{conns: make(map[*trackedConn]struct{})}
}

func (cs *connSet) add(conn *trackedConn) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	cs.conns[conn] = struct{}{}
}

func (cs *connSet) remove(conn *trackedConn) {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	delete(cs.conns, conn)
}

func (cs *connSet) len() int {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	return len(cs.conns)
}

func (cs *connSet) closeIdle() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for conn := range cs.conns {
		if conn.isBusy() {
			continue
		}
		conn.Close()
		delete(cs.conns, conn)
	}
}

func (cs *connSet) closeAll() {
	cs.mu.Lock()
	defer cs.mu.Unlock()
	for conn := range cs.conns {
		conn.Close()
		delete(cs.conns, conn)
	}
}

// drain closes idle connections immediately and gives busy ones up to timeout
// to finish their in-flight command before force-closing them.
func (cs *connSet) drain(timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		cs.closeIdle()
		if cs.len() == 0 {
			return
		}
		if !time.Now().Before(deadline) {
			cs.closeAll()
			return
		}
		time.Sleep(drainPollInterval)
	}
}
