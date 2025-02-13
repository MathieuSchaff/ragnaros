.PHONY: build test run

build:
	go build -o ragnaros .

test:
	go vet ./...
	go test ./...

run: build
	./ragnaros watch
