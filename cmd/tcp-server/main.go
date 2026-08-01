package main

import (
	server "github.com/rider3458/redis-course/internal/server"
)

func main() {
	config := server.ServerConfig{
		Address: ":8080",
	}
	s := server.New(config)
	s.Serve()
}
