.PHONY: build run test clean

build:
	go build -o github-serve-flow .

run: build
	./github-serve-flow

test:
	go test -v -race ./...

cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

clean:
	rm -f github-serve-flow coverage.out
