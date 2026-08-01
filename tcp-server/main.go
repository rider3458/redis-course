package main

import (
	"bufio"
	"fmt"
	"net"
)

func handleConnection(conn net.Conn) {
	defer conn.Close()

	fmt.Println("Client connected:", conn.RemoteAddr())

	scanner := bufio.NewScanner(conn)
	writer := bufio.NewWriter(conn)

	for scanner.Scan() {
		request := scanner.Text()

		fmt.Printf("Received from %s: %q\n", conn.RemoteAddr(), request)

		if request == "quit" {
			if _, err := writer.WriteString("Goodbye\n"); err != nil {
				fmt.Println("Error writing:", err)
				return
			}

			if err := writer.Flush(); err != nil {
				fmt.Println("Error flushing: ", err)
			}
			return
		}

		if _, err := writer.WriteString("Message received: " + request + "\n"); err != nil {
			fmt.Println("Error writing: ", err)
		}

		if err := writer.Flush(); err != nil {
			fmt.Println("Error flusing: ", err)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Println("Error reading:", err)
	} else {
		fmt.Println("Client closed connection:", conn.RemoteAddr())
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
