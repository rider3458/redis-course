.PHONY: format format-check test test-race vet check run-tcp run-thread-pool run-io-multiplexing run-io-multiplexing-docker run-resp

run-resp:
	go run ./cmd/resp-server

run-tcp:
	go run ./cmd/tcp-server

run-thread-pool:
	go run ./cmd/thread-pool-server

run-io-multiplexing:
	go run ./cmd/io-multiplexing-server

run-io-multiplexing-docker:
	docker compose run --rm go go run ./cmd/io-multiplexing-server

format:
	gofmt -w $$(find . -name '*.go' -not -path './vendor/*')

format-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './vendor/*'))"

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

check: format-check test vet test-race