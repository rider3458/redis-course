package server

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

func TestWaitGroupTimeout(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)

	if waitGroupTimeout(&wg, 30*time.Millisecond) {
		t.Fatal("returned true for a group that never finished")
	}

	wg.Done()
	if !waitGroupTimeout(&wg, time.Second) {
		t.Fatal("returned false for a finished group")
	}
}

func TestServeListenerShutsDownCleanly(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s := New(ServerConfig{UseRESPProtocol: true, ShutdownTimeout: time.Second})
	done := make(chan error, 1)
	go func() { done <- s.serveListener(ctx, listener) }()

	conn, err := net.Dial("tcp", address)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("*1\r\n$4\r\nPING\r\n")); err != nil {
		t.Fatalf("write: %v", err)
	}

	reply := make([]byte, 16)
	conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	n, err := conn.Read(reply)
	if err != nil {
		t.Fatalf("read reply: %v", err)
	}
	if got := string(reply[:n]); got != "+PONG\r\n" {
		t.Fatalf("reply = %q, want %q", got, "+PONG\r\n")
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("serveListener returned %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("serveListener did not return after cancel")
	}

	if _, err := net.Dial("tcp", address); err == nil {
		t.Fatal("listener still accepting after shutdown")
	}

	conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := conn.Read(reply); err == nil {
		t.Fatal("idle client connection was not closed on shutdown")
	}
}

func TestConnSetDrainClosesIdleConnectionImmediately(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	tracked := &trackedConn{Conn: server}
	set := newConnSet()
	set.add(tracked)

	start := time.Now()
	set.drain(5 * time.Second)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("drain took %v for an idle connection", elapsed)
	}
	if set.len() != 0 {
		t.Fatalf("connSet len = %d, want 0", set.len())
	}

	client.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := client.Read(make([]byte, 1)); err == nil {
		t.Fatal("idle connection was not closed")
	}
}

func TestConnSetDrainWaitsForBusyConnectionToFinish(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	tracked := &trackedConn{Conn: server}
	tracked.setBusy(true)
	set := newConnSet()
	set.add(tracked)

	done := make(chan struct{})
	go func() {
		set.drain(5 * time.Second)
		close(done)
	}()

	time.Sleep(30 * time.Millisecond)
	tracked.setBusy(false)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("drain did not finish after busy connection went idle")
	}
}

func TestConnSetDrainForceClosesBusyConnectionAfterTimeout(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	tracked := &trackedConn{Conn: server}
	tracked.setBusy(true)
	set := newConnSet()
	set.add(tracked)

	const timeout = 50 * time.Millisecond
	start := time.Now()
	set.drain(timeout)
	if elapsed := time.Since(start); elapsed < timeout {
		t.Fatalf("drain returned in %v, before the %v timeout", elapsed, timeout)
	}
	if set.len() != 0 {
		t.Fatalf("connSet len = %d, want 0", set.len())
	}

	client.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := client.Read(make([]byte, 1)); err == nil {
		t.Fatal("busy connection was not force-closed")
	}
}
