.PHONY: proto build test

proto:
	buf generate

build: proto
	cd backend && go build -o ../bin/server ./cmd/server

test:
	cd backend && go test -race ./...

build-tui:
	cd tui && go build -o ../bin/anonchat-tui .

test-tui:
	cd tui && go test -race ./...

lint-proto:
	buf lint
