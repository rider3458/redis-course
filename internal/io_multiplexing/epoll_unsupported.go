//go:build !linux

package io_multiplexing

import "fmt"

type Epoll struct{}

func CreateIOMultiplexer(maxConnections int) (*Epoll, error) {
	return nil, fmt.Errorf("io_multiplexing is only supported on linux")
}

func (ep *Epoll) Monitor(event Event) error {
	return fmt.Errorf("io_multiplexing is only supported on linux")
}

func (ep *Epoll) Unmonitor(fd int) error {
	return fmt.Errorf("io_multiplexing is only supported on linux")
}

func (ep *Epoll) Wait() ([]Event, error) {
	return nil, fmt.Errorf("io_multiplexing is only supported on linux")
}

func (ep *Epoll) Close() error {
	return nil
}
