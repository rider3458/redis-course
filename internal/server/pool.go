package server

import (
	"fmt"
	"net"
	"sync"
)

type Worker struct {
	id      int
	jobs    <-chan net.Conn
	handler func(net.Conn)

	connectionSlots chan struct{}
	wg              sync.WaitGroup
}

type WorkerPool struct {
	workers []*Worker
	jobs    chan net.Conn
	size    int
}

func (w *Worker) Start() {
	for conn := range w.jobs {
		// Wait until this worker has an available connection slot.
		w.connectionSlots <- struct{}{}
		w.wg.Add(1)

		go func(conn net.Conn) {
			defer w.wg.Done()
			defer func() {
				<-w.connectionSlots
			}()

			fmt.Printf(
				"worker-%d handling %s\n",
				w.id,
				conn.RemoteAddr(),
			)

			w.handler(conn)
		}(conn)
	}

	w.wg.Wait()
}

func (p *WorkerPool) Start() {
	for _, worker := range p.workers {
		go worker.Start()
	}
}

func (p *WorkerPool) Submit(conn net.Conn) {
	p.jobs <- conn
}

func (p *WorkerPool) Close() {
	close(p.jobs)
}

func NewWorkerPool(
	workerCount int,
	connectionsPerWorker int,
	queueSize int,
	handler func(net.Conn),
) *WorkerPool {
	jobs := make(chan net.Conn, queueSize)

	pool := &WorkerPool{
		workers: make([]*Worker, 0, workerCount),
		jobs:    jobs,
		size:    workerCount,
	}

	for id := 1; id <= workerCount; id++ {
		worker := &Worker{
			id:              id,
			jobs:            jobs,
			handler:         handler,
			connectionSlots: make(chan struct{}, connectionsPerWorker),
		}
		pool.workers = append(pool.workers, worker)
	}

	return pool
}
