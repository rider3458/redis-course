package main

import (
	"bytes"
	"fmt"
	"io"
	"net"
)

func handleConnection(conn net.Conn) {
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

func main() {
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Error listening:", err)
		return
	}
	defer listener.Close()
	fmt.Println("Server running on :8080")
	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting:", err)
			continue
		}
		go handleConnection(conn)
	}

}
