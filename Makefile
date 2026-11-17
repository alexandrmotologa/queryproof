.PHONY: all build test clean run audit

BINARY_NAME=queryproof

all: build

build: build-web build-go

build-web:
	cd web && npm install && npm run build

build-go:
	go build -o bin/$(BINARY_NAME) ./cmd/queryproof

test:
	go test -v ./...

audit:
	go run ./cmd/queryproof audit -f ./examples/failing_queries -c ./examples/catalog.json -m ./examples/metrics.yaml

serve:
	go run ./cmd/queryproof serve --port 7373

clean:
	rm -rf bin/ web/dist
