package server

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

type Server struct {
	config ServerConfig
}

type ServerConfig struct {
	Address              string
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
