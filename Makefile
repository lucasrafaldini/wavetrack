VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: build demo test lint bench audit clean

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
	go run honnef.co/go/tools/cmd/staticcheck@latest ./...

# Six runs of every benchmark into bench.txt. Compare two runs with
#   go run golang.org/x/perf/cmd/benchstat@latest old.txt bench.txt
bench:
	go test -run '^$$' -bench . -benchmem -count 6 ./internal/... | tee bench.txt

# Known vulnerabilities in the code paths WaveTrack actually calls.
audit:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

clean:
	rm -f wavetrack bench.txt
