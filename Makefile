VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build demo test lint clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o wavetrack ./cmd/wavetrack

demo: build
	./wavetrack serve --demo --listen 127.0.0.1:8080

test:
	go test -race ./...
	node --check web/assets/app.js
	node --check web/assets/join.js

lint:
	go vet ./...
	test -z "$$(gofmt -l .)" || (gofmt -l . && false)

clean:
	rm -f wavetrack
