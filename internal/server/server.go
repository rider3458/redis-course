package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
	"strings"
	"syscall"

	"github.com/rider3458/redis-course/internal/io_multiplexing"
	"github.com/rider3458/redis-course/internal/protocol"
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
}

func New(config ServerConfig) *Server {
	return &Server{config}
}

func (s *Server) handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	buffer := make([]byte, 1024)
	pending := make([]byte, 0)

	for {
		n, err := conn.Read(buffer)

		if n > 0 {
			pending = append(pending, buffer[:n]...)

			if s.config.UseRESPProtocol {
				respErr := s.processRESPBuffer(&pending, func(cmd *protocol.Command) error {
					fmt.Println("Received from", conn.RemoteAddr(), ":", cmd.Cmd, cmd.Args)
					_, writeErr := conn.Write([]byte(formatParsedCommand(cmd)))
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
			if err == io.EOF {
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
			s.handleConnection,
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

			go s.handleConnection(conn)
		}
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

						response := []byte(formatParsedCommand(cmd))
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

func formatParsedCommand(cmd *protocol.Command) string {
	if len(cmd.Args) == 0 {
		return cmd.Cmd + "\n"
	}
	return cmd.Cmd + " " + strings.Join(cmd.Args, " ") + "\n"
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
